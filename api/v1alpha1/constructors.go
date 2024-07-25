package v1alpha1

func NewDeviceSpec() DeviceSpec {
	return DeviceSpec{
		Config: &DeviceConfigSpec{},
	}
}

func NewDeviceStatus() DeviceStatus {
	return DeviceStatus{
		Conditions: []Condition{},
		Applications: DeviceApplicationsStatus{
			Data: make(map[string]ApplicationStatus),
			Summary: ApplicationsSummaryStatus{
				Status: ApplicationsSummaryStatusUnknown,
			},
		},
		Integrity: DeviceIntegrityStatus{
			Summary: DeviceIntegrityStatusSummary{
				Status: DeviceIntegrityStatusUnknown,
			},
		},
		Resources: DeviceResourceStatus{
			Cpu:    DeviceResourceStatusUnknown,
			Disk:   DeviceResourceStatusUnknown,
			Memory: DeviceResourceStatusUnknown,
		},
		Updated: DeviceUpdatedStatus{
			Status: DeviceUpdatedStatusUnknown,
		},
		Summary: DeviceSummaryStatus{
			Status: DeviceSummaryStatusUnknown,
		},
	}
}

func NewTemplateVersionStatus() TemplateVersionStatus {
	return TemplateVersionStatus{
		Conditions: []Condition{},
		Config:     &DeviceConfigSpec{},
	}
}
