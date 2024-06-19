package store

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func tryLockExclusive(tx *gorm.DB) (bool, error) {
	var success []bool
	if err := tx.Raw("SELECT pg_try_advisory_lock(12345) as success").Pluck("success", &success).Error; err != nil {
		return false, fmt.Errorf("failed to try exclusive advisory lock: %w", err)
	}
	if len(success) != 1 {
		return false, fmt.Errorf("expected single result for exclusive lock. got %d", len(success))
	}
	return success[0], nil
}

func unlockExclusive(tx *gorm.DB) error {
	var success []bool
	if err := tx.Raw("SELECT pg_advisory_unlock(12345) as success").Pluck("success", &success).Error; err != nil {
		return fmt.Errorf("failed to perform advisory unlock: %w", err)
	}
	if len(success) != 1 {
		return fmt.Errorf("expected single result for exclusive unlock. got %d", len(success))
	}
	if !success[0] {
		return errors.New("failed unlocking exclusive lock")
	}
	return nil
}

func lockShared(tx *gorm.DB) error {
	if err := tx.Exec("SELECT pg_advisory_xact_lock_shared(12345)").Error; err != nil {
		return fmt.Errorf("failed to perform advisory lock: %w", err)
	}
	return nil
}

type postgresMigrator struct {
	db      *gorm.DB
	migrate func() error
	log     logrus.FieldLogger
}

func newPostgresMigrator(db *gorm.DB, migrate func() error, log logrus.FieldLogger) *postgresMigrator {
	return &postgresMigrator{
		db:      db,
		migrate: migrate,
		log:     log,
	}
}

func (p *postgresMigrator) performMigrate() error {
	// This transaction is leaked on purpose in order to keep the shared lock
	// as long as the process is alive
	tx := p.db.Begin()

	// The one who succeeds locking exclusively will perform the migration.
	// All the rest will wait on the shared lock for it to finish.
	lockedExclusive, err := tryLockExclusive(tx)
	if err != nil {
		return err
	}
	if err = lockShared(tx); err != nil {
		return err
	}
	if !lockedExclusive {
		p.log.Info("Skipping auto migrate")
		return nil
	}
	p.log.Info("Performing auto migrate")
	if err = p.migrate(); err != nil {
		return err
	}
	return unlockExclusive(tx)
}
