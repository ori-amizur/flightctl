// Package v1alpha1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.3.0 DO NOT EDIT.
package v1alpha1

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/+x9f2/cOLLgVyF6F8juvHY7mZ232DVwOHiczExuMrFhO3vArXMPtFTdzReJ1JCUPT0D",
	"f/cHFkmJkqhuyfGvxPoncYu/isVisapYVfxjloi8EBy4VrODP2YqWUNO8c/DoshYQjUT/ExTXeLHQooC",
	"pGaAvzjNwfyfgkokK0zV2cHspzKnnEigKb3MgJhKRCyJXgOhdZ+L2XymNwXMDmZKS8ZXs5v5zDTadHs8",
	"XwPhZX4J0nSUCK4p4yAVuV6zZE2oBBxuQxgfOIzSVNoZN0d6X43i6xBxqUBeQUqWQm7pnXENK5Cme1Wh",
	"688SlrOD2Z/2ayzvOxTvd/B7bjq6QfB+LZmEdHbwb4tij5gA8mqUjxUE4vK/IdEGgHjXB3/MgJe56fVE",
	"QkERG/PZmenQ/nlacm7/eiOlkLP57AP/xMU1n81nRyIvMtCQBiM6jM5nv+2ZnveuqDTwKjNEB4ZwzE5h",
	"AESnrIaqU+TB7BTUcHeKgok0UaXOyjynctNH7YwvxU5qN5Vkjv2RFDRlGeMrJJuMKk3URmnIQxIiWlKu",
	"WC+tjiam5jSiRDWMdCIdBST0E9BMrw1NvoaVpCmkEbIZTSrNMesxeqsEg/fWiVBJs0IF7s18dnTy4RSU",
	"KGUCvwjOtJBnBSRm5jTLjpezg39vX4lY4xvsWPCUWaJp01BV5HmbcrSjkOkIDoSqAhLt+WhSSglcE7OQ",
	"jrkyRQ5P3hI/vKGlJvka+juvaO2cxVj3uadTzXKwI1Wg1XRqeKEUOcJlSYloQSgXeg3SDGy3wOxgllIN",
	"e6avGGXnoBRd7T5AXD3CeIqrx1cVduilKLWDePs28lz8R+AgaXwZzOwXOWiaUk0Xq6om0WuqW9i4pooo",
	"0OSSKkhJWdhhq4kzrv/+XfRwkEBVbPC/XEoGy78SW14dNtWIL9SgeQ5jFxXBOV5343sa2CzKVbCHCoJ5",
	"jOCq6derH2NCbfACtnMuS9PNDzRTMJrRtPp1fbW++q5bnxs8ooGHALrDopDiynKjJAGl2GUG7R9+i55Q",
	"qbDq2YYn+MfxFciMFgXjqzPIINFCGkT+i2bMFH8oUuoOScNW/OdfykyzIoPjayMTVfWHoeQNlyLLcuD6",
	"FH4tQelgCqdQCGWY2CYKvwG7t6AzybCwmvAPGYDumTWW+Tm+hiuWQIAA+yFEg/3SQcY55EVGNfwLpGKC",
	"O9zYRVyy1U9CfDpMKn7AzJbMGadaSPMht3CZP9/8BkmpDUfasj0aPdYt8BQwC21O/3Roe1fdHiEVO9+8",
	"R6nbLq7hbBwGHEvb+x7XtjWvm54t3NOmK1hRuVJxbkzlqjS0iQdMQRX+b9gi1N3NZ0xDjh10uKH7QKWk",
	"G/M7EXlOeRofrKB6TYRsKCz1OGTJMjDDy5IvyLk5bxPKyaU9Jn0bSi4Zp3JDMpFQDanRSP58cnj+09z0",
	"TMmyzDI7kJuIrR5l5ddCfnrNZBzYlEncKajzWDWoBe81yzIDniy5Pa7ZkjBNmCIZLDWBvNAb8wHrVZVM",
	"J6UyutVa5MEwEQhb7N/jdm7Xs4Z/HHmMk7Y6ZO3pubOTe3ZPkxKhQaW32xMdxASddlER40QeBU3YjBgl",
	"St0lh9elE1PoUoMMiIEmVlRxhKBBIj4gXZAfUFA58Fr1UmSZuIaUXG7IC/UCpQ8FRvZQc/Iitx9yxksN",
	"5sPafliLUpqfqf2Z0o1akF9Kpc1olODhwa6M4IbSD4qkVGuQBur//+9Xe//8eHGRfvNvla/Tj3+ObQEt",
	"2WoF8hiZc7XNt63LDyyD48KLeBEW4OWc7cTsZJl6/CFEHPLUx6LgkjM9lnYd3B9M0zYisL+BZBt0M9BU",
	"dL5u2oesepwSMyoRIWMz6g9bMkiJ8KurKsouQBrJ2xA2Mma1FmWWet4Mv9FEN4bB7g0/nxNVJmtClamU",
	"FxksjJbAEliQt8uasTNFuNCkEEWZIVevSjwEtNSCmBUTVyC9fmRq4alhGH5cK6nmEseNm3VSISaYvDkR",
	"7byJ4C0cmQkuwpPRC6hvuDszXzPl/kLTC/4vCiunuQ+nkAmKkhWFXHD3c5hY6WihGs79DkZ1e8UP7n8i",
	"DO5XDUr1wUHku2sAFtnsX9gR6ox9AVVE2U6ptMjv3kgxbyPpzFGfPV3wCLD1jQqeIBREup5UMLuQSVip",
	"PHJo4XcioZCgUMKjpFhvFEtoRlIs7JowaMGcGN/t8PDkrSsjKSwZB4ULcWW/QUrs3CtjSTWynZ2R3Tix",
	"kC/IGUjT0DOSRPArkJpISMSKs9+r3ip51LAFpfGkk5xm5IpmJcwJ5SnJ6YZIMP2Skgc9YBVzYAppzYYH",
	"ZK11oQ7291dMLz79Qy2YMIuXm9282U8E15JdlkY92k/hCrJ9xVZ7VCZrpiHRpYR9WrA9BJYj+Szy9E/V",
	"AsX4zycWE4Z/Zjw1RE6JrWlBrTHmLZqnb87OKwKwWLUIDJa1xqXBA+NLkLZmtVWAp4Vg3BlYMoZ2rfIy",
	"Z9osEuqkBs0LckS54cKXQEqjBxpu/5aTI5pDdkQV3DsmDfbUnkGZipuzrOFo1+l7jCj6BTRFe43bt9ta",
	"1LrucAuPa+PMOy0+E+wjRwMB+DGWY3tr2E97jOQeAzS1FhKanTTKR92ImKGbpPkLLcxWjZjRLVqifGg+",
	"U9bae2sregeDOM26336c1UJSXKq3MroaLN92rBaRU6+BsohW7GWxrsZJdbI+oXq9RUHWwp8CeCIka8pX",
	"oMwRmiBoXhkxgo9y8hg2NNqy2+iCANNrMApxJeGgoGR0ZE7MnpO3OThDkOcVasN57VqoM2ujckvVsdIv",
	"2YqYpa+4ngFtl6Bu8XI+ylzzI9N2uBMprlgKcpCh5ufyEiQHDeoMEgl6VOO3PGMcbjHqT1oXsWY7cb0F",
	"y56K0BLfkQMKobTZAhGp+ZBkTOE9ydpUQFpDgQ6ceuyM6sEY5qQzG9+qD6DggpsB9mwHgCKsF3cuYU2v",
	"GJJpavVa18k102uCFyCOOakLjko0qhOqd3C/f6g5uowGkgM3R9sFD8X33Zy+xWciXMES7DaMJRECV4vb",
	"wBFsow4kN7vIoudkkcBTo1v1SoBe/HNKXuolTNvMzW43L2mPs5WMlYgZNFenJ0dvnFgTZbQKlOn77evd",
	"lohGX2HLfrjeIl0y3XuTPfA4jPbmzsXunfLOo7Cno8+/Z7e3gNUdO/Pj3M1d2Tbgx96u7+wr9NGgyt6c",
	"/EBZhn/UTg0fuCqLQsjh7hjRkashoqXVuNHSGpie4gDCaubvmNJ9uqAps1qH50f2u5r0wHvXAysG38Tl",
	"u+5CjDgKYofQpHA+vMJpVtGqm2PUQL/U/Wzs+Cyu0rA86tUhlJZghBy68kZfST6cvtt9ItsOtwLS57IV",
	"B6UlKRyfWag+H5LqkrsHnqQoh+2dZkded0iZ+vQ57XPIxdBjP9ZD+8avKGdVpw66objpdyf7v1Q6d78j",
	"yTRLaHZrx7LYwKHfWre0HjxWGgAUK/ZAxspC95HAntOlEJRSR8naTsqu3WK7veZOATanUtPWEF7I0b3f",
	"P5p/Xu79c++/Fh+/iV/J7RTmxcDzwfEP64/rzqbucW7Gcf64lu1760NDjxp+NrVM31FNybtpDEdjTn97",
	"B3yl17ODb//z7/M2Wg/3/t/LvX8eXFzs/dfi4uLi4ptbIrdz5Y7E0r/naj4Uk7hsaWiDj0vVzscP3Ruc",
	"aZ64tuaM1ZKyzF84lzSrvRLpFkt+bWkbRi0R46MlemtnVFu8KoMpIpjWF9BdoyGYUZ/KEPqhtjnn4Rl1",
	"PRm9satZVhrNrXSVkXuyatPYlWMPjBFWV0eMTXur34VvnTI4oIO6/s185iS2YU0/2Mr12K71ISorQ7xZ",
	"uxvTk2VjIvMm4Yc4Dle5ohZcuHoyNUpDELds//t3aHdWFu8GfHcK92d5sfd1EQgcx2jujLuvn8KlEM7R",
	"8ERcg4T0eLm8pfjRgCIYtVMWABIpbQoXjaIQ3EhxYwaR8oho0th60aOjquH0NkBFkaVqvyxZivpwydmv",
	"JWQbwlKj1Cw3gVkuciIEylD8+uEwqGE4OhoXyGW72w7VGeRYS1uzz++F0OTt6zFduZt5vrLzj8N57CsR",
	"W2v4AG39LERJNY8uFP07oMnY7tzS5ja/ZUV3ufkbcN9u83e7CDb/h+JcvKbaYPW41MdL93fgX3ybnd4Y",
	"MhgiUhqOGm3ccnRuloYblqlPj+0KYhQ/UiqnQbeuanpvE2v3m9i9YrPP7fukiN/sGfR03Nu7sHSqNJ1S",
	"nEUIgaLoG08zvFXCZltF3MlIOTmrPDtnlc52Gue30m1+CxcWB2nscOiJd6FZxDXDR8J0aM6X+JA0UOR6",
	"DXj5a+jCs4w1VeQSgBNfP2Bll0JkQFFT9KWHun+kQ7waMZ1jZB7VgcOiH+6aqsZIw6LwfIvvN/2jf7/x",
	"o7eCuU2pjJ72Gb2ETG3zBOo0aY5tO2hIl+6TFnhfv/HsrCNO9VhLqvUcRBfxq6poteatVafKdDQ89v1V",
	"dEkGmXS68sN0qfWVXmrFD67dHMBUs+scVLT2w07dF4poKlfgrIxdzpCoiLN6oqQd4OTNL3vAE5FCSk5+",
	"Pjr706uXJDGNUTIHotiKG7KSNZVHuGzTMDzcifQOmPphm5X7wGrnJ2U96QPuzpRXMq/XwImhZqiQ6jy3",
	"fLjqDlu5kgOXvcdm3lNxnPl80OFQCySjWFMlydzMZwFVROgpIJkOXRkagjQkqygZbTW8d7MTwGfw4C1m",
	"9X6za3Sp0YTWvdXpy0OA9X36gZ1aaBXQfjOfNaPQouqv6czgpgr2sJvBCHFVxhkX17NkGS6Ct14cSbCW",
	"g1PIxVVluLC2hAxckfOJPUIHwxOQOUPPMTXQpNGYQjVi42s1fONrBUvjawVYqwcHZXO0Lsg3Lh68i0v8",
	"3FTZHY9JJ/ehSTOfNPM6NNbslHHauG1ytxo49hnXrqqipkaFn6d9/OhqVL0Ow0KxkWFP+tJXqi/V7CS+",
	"j7foRUtTvlMXUi4ZzM6pGV3CZ45BenMpX2Ki3kPE6LWvquKcsJ1xwAPdj+sexSQoHKeM4DIMduXB2nMC",
	"GLFFs2xDWCVjBTXIml4BhsujQ1riw+VzyukKo1sqZY9xQsn12km3HT+4cfqFncyD6xSYT4wlrYimURFh",
	"MQfA83iWit7Yvahj2LlNYoFNtsB+CoWo7gyjev2SZgragA5JGOa79lMtZRbXhv5SCEwjZc7GXGj4K16a",
	"2+RT5MPpu53al+nZ1YlONRpPN/iStLvKN/NO0A/Tp6aHP3puQCNZSP0MdyfGCLBRH32ClAoIVS51Bk+I",
	"LcHYsa5vITLbU7hiKh4Y2omDqsDrNJ733bm2g5csTuJ3s3Xc4EjKS+gikRHx8Xuq4O/fEW/JkEJocnQY",
	"w0VBlboWsjcRlC21d76lXtvovp/Oz0+sk0MhMCtFdcFSdRdze/jECiuM/AtkdYXeHfjsEysc8SODBGmE",
	"1bpB7OZIZ2oQJs7fnaFBh7hDfRDgpvNPsBneuak8tG/xCfrsIqboTjBfKpD9aWd86a6hupukw1x6AmDv",
	"lLsY0TLKXpYsgx0h2/6EZBmaUSU4lqIKwRWaAJUW0iYr8xVdhH0jlnIRZywPzMdUuVyy37pDnVBZ5Uf9",
	"cPrOmtMSkTeCby+pwtIFeasxIp3xJCtTIL+WgJ4okuagUdeziYEOLvi+QeK+FvteZ/jfWPl/YeUYjNsY",
	"abVcO3mnX/F+5nnLg3vd4LvDIruHZggdfODjPsNlEiShWUaEJEkmOKCKNua4n4cTip39vYHtd7pBmfXs",
	"7F0KLUvYteSuj/iKbw3uv9OpKOw/ym1yUXJ90ifR9CaWQKtyQZMBoqszCNct5sGgOzdNDXociU1dMRIL",
	"n9sEI59gM7f2h4Iy6S6qqARy+P41pAvyJi/0Zp+XWWavsohXVo0epZO1UYDWjK+6ig0Wvxt/kbZ93mGv",
	"sT1Qqf9R444pcVr6JSjitWQ7a7Xheg2aJXVOAJKXyip6c8dAGV+huU6hjeuKSiZKVSmbCIZakMMgiodu",
	"rKYoeLbBTNJiSf6o9e458YDdRJVDzXgZu9pyJdj/JeBVAFtWKbnwNyUZy21+Od14WAA1RyJBl5JDOnc5",
	"HryHT+OuEiR69+RCgs2XQK8oy+hlBphDwtmumCKioL+WUFn+LhGO1HA9phQW2KQR3onH5zetzVPUKsyo",
	"RjNljaJaGDAlgyuXfhR+0/7ao4KkxvuRxYpZJGrUcsWUNgo09mXAchYup4SBR5mbaTNjhZm3zVqREvQD",
	"RXmCGl1+Cdc+S6Rd3AIDzC1K/NJ7s+ySQZZW2Lb3u6WyVj6mSLWSFpU+FZv1JU+sD6auMe0lF4n+m1ay",
	"mZOSZ6AU2YjSwiMhAVah0omaUuSYdSa87et5NiKnjDO+eqshPzJMqUuA3TqV61RFZ6q8VGa5TRmSnIMe",
	"l6N+0sIsihNPnGjml99PsMpS6L5aEvKBgKljTUI6XFc8am4atam/gtwDpUhpHY2Rei16TTd+KTAHXslx",
	"S/GUiJxpDSlJS7TeKpCMZux3+05GA1BcXfsIA/mLywR0CQk1UqBNr4fmo3XJP5meRF2KKHD4RA90rPTX",
	"ej4SHOosXbbnZCfC1OfMxFuWRZaiUEk5uXq1ePWfJBUIt+mlHsPSPuMauFlGM4lKFI5RyjegNMvR+fsb",
	"uwfZ784Al4jMrB8CcYQW6+pGwowrARlpX99aeH4opPuBiTEHpa2PaT2/YNzv/byVENhfOzusLjP4ap5V",
	"RpAsDH9RZv2i55XdX25fKWzh+KRL2oh1E7zwjlw5cS50HbZ4SzeYurJ9Q2AT+sBEE5ghPC6LvtI0L4YG",
	"ppmhM7hl09WWxxIOieVhScVDGjc1QWav4CGFSp1URnBxhn9yUuVV9ZhA5XNBToGme0ZAGPi2wmf7J/n0",
	"cvYC6hNsvDyTlV4CcEm//Sku5Ipys0VNPSMorIQ0P/+iElHYr5bt/rU6jmPrG7dThJqzqxsLSrrmEJVl",
	"g0syqom45srfddrvRngjF3jps2+GupgRi+S+R5PC8zsyIPfSjsMfDuuCv5i7gLUixQsV3I3WyRjqK9dh",
	"hpcTI/UGgR2V6X+ENiyKuIIa+OVUiXZDJxyaphi/WWRWSZHWGeZj1NoYM88ckv9zdvyenAjERH+OYCS+",
	"OIxW9tGC0BRlMQfNoqMeYFbdoi8BXvt+9tRlwIola4tne+yC1kq71c7n1sXRlNJtbEq32I5oLt3npJjo",
	"JQLr/OcTnw2aBla+dYqKB0pB0cku18sBv9w0FXeUcGI+KElewxbYwVhYWsW2OJ/IpqU4YLorpp29L8po",
	"T7dYok9Dy3PgLPgj06FV2mYgResk1Fn3Jr+jyX/w2fsP1jtonBNh0O5uPQnrjuPuhM3ypk9hVcYmD+HH",
	"9yyUrdUYeERW3H5yMvxKnQxbPMfoa8PyZ7dcm4bksB5c+Uyt67o7oO7x2WvXGOe4V8srg733giaf72vX",
	"7OxhHe68YHyYgdSnZSz9civ/fVubXJc55XtVfpOWdyqiz/Qdj54r++xhrwPds4rTFlfNB7muQNIV2LwW",
	"eDvkI30uYWl2OA7M+OoeH+hqPsJ1cZH+x5b3twqQiTm5Vj0GiLrcoM5Oy96T2YezVBSddk42D/YVDMlr",
	"1lj0M9conhfG9xisVWMeTZPfTgprDBZkkIlmqcSMScPCq3oHqTvurRKM2FvHghLMxiuSO165PDr50LuF",
	"4y8j2xw0vXp2T34af3/Q167/dqH7EqZTtce9sNAzm128fxtcOywOPZi4iaxSz1shnuVtM0BgJSJLfH3j",
	"2F+u268F3oBbIkEpyDKV0UaJmvdGBK9wNaK572leZIyv3hoR1sW59rDSS9DXALyypWBTM68v4vnCtotG",
	"gJd5uJYRlGxjS2cbnsQEirq0ncBoCRLvabSwjhbu0h7d/KwTfmAA0cK64KGLgZN/Uc+p8hlOqtJkDJmM",
	"IeFz1iPNIUHLuzaI1F17k8i0Wx/XsOHabngy+phFTj+ZNr5a00aLg/RGDvW79dPq2bTGY6YtHZ28xWTW",
	"vsb8gutG/sV6j2rKuPXIjJ399kaUiwuuykvfnJkd+IYmawtKqy/r7eF7wHwXKIFccOef5V+ZehKhBd0Q",
	"qUjaSue7Il2tLr7HBQQMjaxqEUyvXaldZ6xlqeZXn2cnorfjfVtzqHtzyZHIc6bj62PdArECWVO1rtOq",
	"GDggja+87/nHLR5PVe+BQ1Os8yHedCMMXmdqfasouUKyK6rhZ9icUKWKtaQK+uPdbLnVnNT6pGr7FMLc",
	"mgDtikdz8yZnZz8ND0m7iSP+lhE2KlyyHZbke4qvMbNvXW37aJtbRtnUk4pSaQ9DckyIWU1Ul5I7uQQf",
	"D6WZTwKWCv5C+xrW6SbwlxuYsGmIbbfmdlb08W5ePT5vVMWNyDlN1oxD71DX601rAIMDd1Zc4JtupYSL",
	"mYPHOUgzVUcO2NfFrU8zukQ32Xcdb3BIThFMkmRUWk8778LgJms2BrksDZbBOleLK5CSpUCY3pGsO7qc",
	"3iexQh45xgiOA3IxOyuTBJS6mBmxJJjpvUt6Ri3aozzdc8AP2uTnLu3C69Am2ngyPx46viMea0vUWW+8",
	"6DDDcRTgCsZZz4wawPZVCkHuqxOEBH4M0NerVLYqNE1Toesn8Qkwptv4ycQ0mZio2m9tnXFWpnbjuzU0",
	"tXqPu99EKjV9cFoVJj+cRzdXxVZkkNrWPgcmq9VXarWKMaVuTop4qs5zn6aJXK+FgurE9/tziQ4DYveD",
	"H7b/IeBVvHJYQFqYxWu+g5/dxrxSzdhxqTvwxbnLN+ru8NmzWPj9DT5lZ58cylgC3BokbOzT7LCgyRrI",
	"t4uXM6fXzvzOur6+XlAsXgi52ndt1f67t0dv3p+92ft28XKx1jlmPtZMZ6a74wK4e7iY/FLnHjs8eTub",
	"z678oTIruT08Uve8E6cFmx3M/rZ4uXjljHGIU7NJ969e7buEZ3ZxMKVtZ5ns90bAZvCIcv1+k+BvU3xQ",
	"y1SvS31wL47x7cuXPuAdbLhx8IDc/n875dQu7k5jg5cBOmFvxz+b2X/38tWdjWVTLUeG+sBpqdcYI5da",
	"jYyuUK+xiEWlYhVjHig09OHQ8Lm6rM7ighs+EqVm7Th1uhdzqtsMMN4sXWY6ODespSoM7Xe7D3swHWDU",
	"qE39oNuVXvhY9hcu7tiZAQoJV5gnoRnUjU8Bzg5mCJDP31anNjByWbUGnf0YC9O0Ud/uRl9Llug6Fhvv",
	"qFwIvo+DtVGYTLos7QvyGpYUEaIFgSuQmyq3RQzQrJFjYyS0S5a59YjC6vMNukDRBpptUxdWWiryCTZj",
	"Qbctf8COGpAPj6CJHXo5/Y3lZd4ItrcUVuE+TAFQh/ef10kYMFbdxpb3U1SjOWHLJjnDb0xp22kruwJ6",
	"j64BI1td3C6khKpgh6CfSJC5ADHXSwIsx0idGoGhUfxv30aN4ndKuhgUO3b5bSTtNor9eI/82TIw1KW2",
	"8OiX98+jv6cpCZ47eYRzwQz6t/sf9L3Q3geu7ywqREy1tekBCHUHUuc8svnyq0KnWnwv0s0dU4udVS2D",
	"aVnCTYdGX93LqC3hFKecPjMi/ef9D+qenhd8mTEXktmh05t5W0Dd/8PwtJtBcmoPEYeC6S6pKryIr1og",
	"i8Xr7IrDuuxeTYJ9XIb7pARiM+h3D8L4fhAlHyeBS6A2DVAtIfRQzinQdBjd2DeCyUQ+XxX5FEYPimXo",
	"1Mna5wGpaCiN0xBWHs980junnqFH9x7O+j/GobiRweTGHeaPRq/P5th+CnukjLJYTOAylMti5adwQD+u",
	"ePtwW2QSpb+SPfklyO77QZqdqEDm3yu3OT1FhmYdbi3OEW6BlX02nq9eLqvSDk3i2VB680l9eglu5cyP",
	"yzLLqvx+9r5rKeQwue5H0JEkVTvI8f19SXjzXidfm/m0necobjfEuqedqo9D/hHsbjnPvuuu8ntBPCDT",
	"afB0ToPa76dfO1cN98wRevqZd5mcrDyTCoIqyGhSCpSRp0BNz0UlmTSERxGd6lfIvd/YLVxC6jex+9xC",
	"Oq9mP2MPkQ7KdziL1LgjAfK6jiNRHE8+JF+qD8nkcDHQ4eI+ha7OnprcGoYws7i3gX/Yo25jvUm3Oh90",
	"VuCe/BC64zywS0IPAL0m1W9f/uNhxz7MjG62wZSjcnKReFjFOrbPtopxYxwnuhLGUDFujG4UHeWpa92D",
	"dsazVMBHiLERj4sar1FrzmhCs46zfAWykIzrLs1NJPe1ktyIG+gBjM4ZgO6I090D1T0Z0edRKP4xJa7J",
	"RPUoO3yImLNPi0IKl4Vzu6+zq9i1CMd27SCN5NCP/YxYRDXnx2YVTUAmy/KD3jZ+++1DzLKQIgGl6GUG",
	"b7hmenM3LONzLiJ384qoFDv+QmkSYJ+5APs5FBiXZJ8YET5veXbaACGzxoQIt7mB/ME2jFutqsJneuHo",
	"0kxsvWTsQeA7pnRVNN0lTneJU/D21x28jZt9uuTsY6A7wqgRez1mA192HxKP7fuBLyyDQSeT2WPfD3oS",
	"7QhT+3/g/zf7PmeTyxl0GymrnfapT+Bqp1/bJTvgO+WG7fmTvTPQIq5xLIM99fh679OWAlvrv0Me3L3U",
	"5pB4wgs9nwTUSUCdnN3G8JRYNtRJCtzCQIcftmO8cdo8cdgh+9ms9/44b2hKHDjqk7Jnd5LCTsa8cRJF",
	"xP9nJ5GfAk2/HBJ/P5H4MyHxCM8fztrj9oHASj3mVsY3eOq01WsneD4U9UD2ga2WgeG8OU6lhiEPotFI",
	"zoWJVL9E5heYPcckwlpGyQfrjuZxy7smnK8mC9ZOUp2cnh5uewz3QO7jrVj38UWAR72aeLDNMd2CTGLV",
	"XYlVffrAZ7kX7pDAxntwTQLYV3zCjKWi+qx5AoT0PE6cZ0q4AXOsHnBlt3p15jRsHjegtKo802ve4FHu",
	"7Te8chtG3zGlW/icvP+my9XpcvUz0hn6fTndq27lWDtc7ILacT+707DCfcgXwQAP7HHXHnlSOB/b7a5B",
	"uz3SzpgLoi3U3RJyNmOk9ka3T10H3E7lz1KeHiLURS5ytlDTKdB0oqWJlsZd7WwhKHf38XQo6qu56RlG",
	"w5OF+YH3zfA7n61sGBt8ifvm/gTmh906k4D+DPZrQzS3j++rDU9uZ4m07c82POkV0usqz9oUWWN6pzEy",
	"qBo3RjawPhkjJ2PkZIz8jHOq3k2TOXIH19ppkNzCurxJssG87kfGCoZ4cLNke+xJ7nl8w2SDivvkn3G2",
	"yS2E3hV8xmkyja6fvlVpO8E/U7vSEGkvaqXcQlfWTjlR1URV/jQeZ6/cQlrOhve0aOsrsloOo+bJDvLg",
	"O2iM5XIra3a2yy9zB92nbP3Q22iS5p/J7g3keC0+Ad/3aRT73MyxFpE9KULPTWn4rk5AxX+ziG4/1Zwy",
	"CYmpvAaa4i7/Y/ZOWEw0kdDenQb47179o9vpYanXhAtNEsGXbFVK1Mi7c72iGUuphh2TddViQeU433/5",
	"bjrMCnmQnVfNhQx0wLVb7NskZmsZwGogA3qO1WG8rjUGbzfzmTWS2VmVMpsdzPZnNx9v/icAAP//q2pG",
	"3ycxAQA=",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
