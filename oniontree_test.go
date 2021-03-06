package oniontree_test

import (
	"bytes"
	"github.com/go-yaml/yaml"
	"github.com/oniontree-org/go-oniontree"
	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
	"io/ioutil"
	"os"
	"testing"
)

func newTempDir(t *testing.T) string {
	tmpDir, err := ioutil.TempDir("", "go-oniontree")
	if err != nil {
		t.Fatal(err)
	}
	return tmpDir
}

func newOnionTree(t *testing.T) (*oniontree.OnionTree, func() error) {
	tmpDir := newTempDir(t)
	return oniontree.New(tmpDir), func() error {
		return os.RemoveAll(tmpDir)
	}
}

func copyOnionTree(t *testing.T) (*oniontree.OnionTree, func() error) {
	tmpDir := newTempDir(t)
	if err := copy.Copy("testdata/oniontree", tmpDir); err != nil {
		t.Fatal(err)
	}
	return oniontree.New(tmpDir), func() error {
		return os.RemoveAll(tmpDir)
	}
}

func readServiceFile(t *testing.T, ot *oniontree.OnionTree, id string) *oniontree.Service {
	bytes, err := ioutil.ReadFile(ot.UnsortedDir() + "/" + id + ".yaml")
	if err != nil {
		t.Fatal(err)
	}

	service := oniontree.NewService(id)

	if err := yaml.Unmarshal(bytes, &service); err != nil {
		t.Fatal(err)
	}

	return service
}

func TestOnionTree_Init(t *testing.T) {
	ot, cleanup := newOnionTree(t)
	defer cleanup()

	if err := ot.Init(); err != nil {
		t.Fatal(err)
	}

	// Check temporary directory
	if !assert.FileExists(t, ot.Dir()+"/.oniontree") {
		t.Fatal("file '.oniontree' does not exist")
	}
	if !assert.DirExists(t, ot.UnsortedDir()) {
		t.Fatal("dir 'unsorted' does not exist")
	}
	if !assert.DirExists(t, ot.TaggedDir()) {
		t.Fatal("dir 'tagged' does not exist")
	}
}

func TestOnionTree_AddService(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceID := "dummyservice"
	service := oniontree.NewService(serviceID)
	service.Name = "Dummy Service"
	service.Description = "Describe the service"
	service.URLs = []string{"http://first.onion", "http://second.onion"}

	if err := ot.AddService(service); err != nil {
		t.Fatal(err)
	}

	if !assert.FileExists(t, ot.UnsortedDir()+"/dummyservice.yaml") {
		t.Fatal("file 'dummyservice.yaml' not exists")
	}

	serviceResult := readServiceFile(t, ot, serviceID)

	if !assert.Equal(t, service, serviceResult) {
		t.Fatal("saved data do not match")
	}
}

func TestOnionTree_AddServiceErrorExists(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	service := oniontree.NewService("oniontree")
	service.Name = "OnionTree"
	service.SetURLs([]string{"http://onions53ehmf4q75.onion"})

	err := ot.AddService(service)
	if _, ok := err.(*oniontree.ErrIdExists); !ok {
		if err == nil {
			t.Fatal("service added even though it already existed")
		} else {
			t.Fatal("unexpected error", err.Error())
		}
	}
}

func TestOnionTree_AddServiceErrorInvalidID(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	invalidIDs := []string{
		"",
		"Dummyservice",
		"dummy_service",
		"dummy service",
	}

	for _, id := range invalidIDs {
		service := oniontree.NewService(id)

		err := ot.AddService(service)
		if _, ok := err.(*oniontree.ErrInvalidID); !ok {
			if err == nil {
				t.Fatal("service added even though its ID is invalid")
			} else {
				t.Fatal("unexpected error", err.Error())
			}
		}
	}
}

func TestOnionTree_RemoveService(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceID := "oniontree"

	if err := ot.RemoveService(serviceID); err != nil {
		t.Fatal(err)
	}

	if !assert.NoFileExists(t, ot.UnsortedDir()+"/oniontree.yaml") {
		t.Fatal("file 'oniontree.yaml' exists")
	}
	if !assert.NoFileExists(t, ot.TaggedDir()+"/link_list/oniontree.yaml") {
		t.Fatal("file '/link_list/oniontree.yaml' exists")
	}
}

func TestOnionTree_UpdateService(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceID := "oniontree"
	service, err := ot.GetService(serviceID)
	if err != nil {
		t.Fatal(err)
	}
	service.Name = "OnionTree [RENAMED]"

	if err := ot.UpdateService(service); err != nil {
		t.Fatal(err)
	}

	serviceResult := readServiceFile(t, ot, serviceID)

	if !assert.Equal(t, service, serviceResult) {
		t.Fatal("saved data do not match")
	}
}

func TestOnionTree_UpdateServiceErrorNotExists(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	service := oniontree.NewService("dummyservice")
	service.Name = "Dummy Service"
	service.SetURLs([]string{"http://onions53ehmf4q75.onion"})

	err := ot.UpdateService(service)
	if _, ok := err.(*oniontree.ErrIdNotExists); !ok {
		if err == nil {
			t.Fatal("service updated even though id does not exist")
		} else {
			t.Fatal("unexpected error", err.Error())
		}
	}
}

func TestOnionTree_GetService(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceID := "oniontree"
	service := oniontree.NewService(serviceID)
	service.Name = "OnionTree"
	service.Description = "OnionTree is an open source repository of Tor hidden services."
	service.URLs = []string{"http://onions53ehmf4q75.onion"}
	service.PublicKeys = []*oniontree.PublicKey{
		{
			ID:          "E4B6CAC49B242A44",
			UserID:      "Onion Limited <onionltd@protonmail.com>",
			Fingerprint: "F01FED47979554C92D9F56B2E4B6CAC49B242A44",
			Value: `-----BEGIN PGP PUBLIC KEY BLOCK-----

mQINBF2m6moBEAC1qYRGWYi6j8cOLDIEyGvACVbpvkAvFR/rAh0WfSPyP/Ch3ThD
5J1B9bCYHJZuhfn/WSk+/rG5rxY+iUv6CaptM3PPJBfhWgDuTOjircWljh2XiMO9
PyAvJT3+woQ20CJKPoLazAa2W7rBLfsuC0y4JI3y44P5b0AxG3+xaWryWb7oDuJm
XCOikPWWxMnun9bczH6Iyz9auBaR4Tg7cewLhPvMcGI0gVL9Ldyw4OYnVqDp4Vnb
QaKaeza/XX+rZ9EtuGdM7yhk6FpTuNi/b4ZAZdP8O11EQTKRNhvJ5InsUBpTXPaI
FS2+bsXgPmGj2Sb4HtZGV4KYmdtCPbdQ4CIrK+8Y3dVQVHlSbvfYS1Q9VLXW9nB7
Lv2rHNW5bz8IBJpvQ4Chp64mtqOnKZ5zRDK+dH4hXdhRFiWci/JjrmQiEB7UkK+a
OvZIFzjkxN0X0yM13bN+bkliGbyyZPgxihQ+KjicLXcCAQCaTI2FM50CDUduWNB1
MIJykCs3t32F5ivmaVRaZYqSDUIgP6ba3NvK66L68crC4GM00Rtc4cFzkVkJqzUX
o900lhTj2epyWuFy2EzBhPgLmojYAKX881e7osFqZ1WpxiFElTxo5uuhhN/WMiGI
clt2EFJeFI+z3DnrXrtugH0xpA1rkSuBIAzmN4nlKE0hBBm2oyIskW2HqQARAQAB
tCdPbmlvbiBMaW1pdGVkIDxvbmlvbmx0ZEBwcm90b25tYWlsLmNvbT6JAk4EEwEK
ADgWIQTwH+1Hl5VUyS2fVrLktsrEmyQqRAUCXabqagIbAwULCQgHAwUVCgkICwUW
AgMBAAIeAQIXgAAKCRDktsrEmyQqRCtoD/9/SRzyuXPb3PynYnIANx6jaUe42jym
rgUkOTm6W62KF2U0aNe2BuISfelhf2e+UHTf2f4csntkZEFbjQ7GHCELhoudcd1o
+wPEcboUY2N0luQj279LCnK8kAwXHmQcgntR3wUufG9Z+dLTDRdwZWtt22cMb57z
O+Vx6rLi/ZKuytXXl0LvYJoYQN6c1rp7nvN+K6QlbabcEsK7ir3gr7/IcJkYjb8g
f+91ffJ90lgHh25qWgZ80SzixHprgoZf1qPIl387wAjQL7lTHq3ZE7HvQl6pVH2e
rgSOPc6Ny79sRWD66QAsQeZ/OzpjPmkekorOU/PQclsqYZHMf/UGFk8+ipYKAdke
Oj4IQCJjyQag4ZXYso9EC3MWWC7opRsBmb4iFtwLgQ9x/vONg+twmECEVvm2Y8kg
lPV9zBHVy9qJT+1lBYvXv7E7Q0oZlEU9Bsp3RfFPG//71ngYh2bzXx9IXA275k4W
e3VACqVEbRz8+e5b1P4BCj1e0PQKpJNBbIiWoT1lT0BnX6xYHX1bH/IY7hssF3tl
XhxQlgCkiLL3eDovIuUIMCVHAsn+thd2AWV9gJuQyW7uRTqYBkwY3XaXRyGGJcs3
cXJ6bDeiTyyMIHDDZ8acrHpGG/oWFc43+qNLYGEOssw3TYnOTuCW0qb7SXvCMYvy
e1nerZmMocdeM7kCDQRdpupqARAA7sWgYE/kHXx86Lay8NWvej4LnMdgX2p3EXdo
RA4If0uIK0LnsBlABDL9qxXZpML2OPmljkaOwEwbtZ31JmbP+8Bg9mPwvfOP3UZL
1a7KlC6zQBB5INxrdg/aSLPlvifhKe8qDgr5/h16jWb5dlPvPtZT7lbymTUfCaaA
ZvNCUhpalJlJA2dDxml2w6jSBRCuXJGsHZA+CTidz7KTCOiN9GUCNn16bJqAGWaG
hQFE8CI6YqtnyvcmdT1a3ZR34QLkOrxw6RhAMxf1iUsAvebi8ZNbHgIlF7QxUy2z
/A57zRhODUtwvpyA/htByD80uvxu/sxmi9W/4elPd08e6JE5VUR1CNGqvCupwehl
qxWKYPjkaxzbqrl/+V89v8L0nELTxZoQ5qSGnSlR58YRrVfrpbMZbmVQ7cLWANVA
1htr+MTi8A7LftFP/nFrqAPqPAlWESJtuFQDRAdN5sA++CNubDMGKzsa4jtCw9TQ
w9bzEfKthqPShfSsanBP92gPAqWJ9WPprh/KCuJ7onwGNc6JfheIs9DUNaemf4WQ
zhqcwme4fnTwn6+Vjt4uqnCWnDHQZksArzilhY01PjrMSWE5ITbI8Trhc6LCIF+j
wv63XEMdLDb0oYhiZnklL88TOcTRGYv9krycEZRVWQfQ5nZXKa1ufJC3hKSdW9IS
fNuiGjMAEQEAAYkCNgQYAQoAIBYhBPAf7UeXlVTJLZ9WsuS2ysSbJCpEBQJdpupq
AhsMAAoJEOS2ysSbJCpEN+QQAJgZRL4aNq9AdxCxqYR0bgdNGKyFqYL3QqRmANW7
3suJDdCWEfg5TomcaH6BUi72V8U3q8OYp+D/iprtuumGZvd1+RImoOIA4WzpC7P/
A180ShXDvlMRjZVAoivkL7WCV87TbArsqGdQXMWF7jnM6MNd/el7xWfZx7hIiS04
YQIdlEJR1Zs1PYcNnk2EdY7r/JWQ620MO3SOvLOqOg9+4wtzw8h3G2AHar1xdi1P
qD2PHSWhyVqIFcgTL3C7oAJun/GNhlm1EA0PrXaw5pwgB1YAT5aPZYWrD0TB9Vjv
EdiE27fCLQCUaMjLQE+OD54fGUqXksznilpNX7mh4/a717bHxQVg+brkdf1NXTDQ
Kvl8Tvc0WAWW1EZSQ1BRO5ITRV2SYQ2K5CN0P9tSViuLAwZnPOlEVGoDrpwT8Ea8
qM6SmAT/8Q06CRnKed3pPLOp6f6v/97/sK/J9l/aIMiaJFO1uR2pwBkZBwqUn7pE
PScW4PjEMUaMaWednXy08LoqvJ7YovBS0EGXnGUmAFUqsalIJwZCV0MylMKzq9IA
jPyk5bZSeMlli9rUUdRZg9KGNEvUwRzfyqRyquYMb9xgOjWWUlw1nDRcF/PqSY9/
ObwNdRKHOtR7g0DzpvQevN3B4/IKbpA/v8MIKQjD6mpGDUUpoRc1xnudhwNwjGYN
QlyruQINBF2m6rwBEAChj7qqFrycCCBTn+4lycxv0a7AyLi3nlp95ZbzJCBM2lcI
VmVOE4mlQ3pW5/avID28NnOGtKIc+sEc6AE+WW6DfT1f88wquIbS2H8QjUUwXx3n
RiE889CSxroNJuXrcVAqgzwih6Q2Pm/JFEWcwMPefGddF1KNvyWciB6IcGpZAaYP
u7nCgdYF6dgPQPSn8ylRFMee/31Qr4wLQMEB8UF0p5nNmgK9Cl0J7h1cKAjJiJt1
gTUOOakCeBEQ3WrmioJ1Lvkv1YHVGQHVaqQ2eYcFBZA+yIYqYDsIVB3SalzM/8OR
k6KwJtLficVyv1gj8RyNcOEhFikkJuRmzxnhf4qxfj2gNGZ7ws1MxxMYvtslvRel
1ktZFwFFkpfh5f1DxxzPD9EVjk5Vc9ykwnxNr7PGwZnh56Bxr662FIAp01qNOVs8
1celHL563m5VJdIdEm6YbCx9gqR/FqpqPsFRzQs3B2vQG2FlC6ctGLfD4e5jKlD9
N+ZMvsqvj8NPzFqE6JPQ0VpODhiLVF2HBgdf8FNEkiJNw0tZqZOPnMgwgNeDH1re
i+S28bH5OJEAEFb+lTWUmp6wxInMPhd8p/KoXF90MIrKkvbOEz6TNNHJrihCJdHg
JAY+Eg7VE6lsYktmS8VQjsenOKLZeSBdWCBjcA67mSoVQelw5kTHgLlNJjTDZwAR
AQABiQRyBBgBCgAmFiEE8B/tR5eVVMktn1ay5LbKxJskKkQFAl2m6rwCGy4FCQHh
M4ACQAkQ5LbKxJskKkTBdCAEGQEKAB0WIQQ6U6ImiWaHu2722SrsW7I7nqStwgUC
XabqvAAKCRDsW7I7nqStwmwaD/4rltArtT0q/+NxQZ+wFQeCYUw8lm7PwXqmOwnF
bnkyqsrwX70fluNWQEJXrFfc5gQ9OnflDnQFCn+17ifU0XHysHHd8OzisyFHKzPC
PRwCaoN2bImxgK/ZjjeFTsVWBTwN/vcrZIbaBlVzSkxPU+MRdrTXlfPOr58onqwO
rse9mzCLUiny7NXc2oUd/xfKCnoL4YU6wP6/Qe2Vcxu6YIsMzlwozc0DnMqmM7gF
q6dYYOJmWqwIyFcJ0wgqNdM5IdItt7Dt0ek8ry1Mva5BPgnp5D9K9twTAVsoex0/
uLLRZ8QkSTJof9PlinobGR8H9wv6AsTsSJn1BvVdu3yNQM7FSo7e8ChZSaTWdprS
dz705o2RVNodQaU5HL2OLR8mWmhsXuDL1ff6cDN8eO81emONEfBbDJuHFFUwylWq
h5DOsG2VEhz+NUbv2ZHfw4aK2xW1aAy9qeKZ7TuYfKfXm6rLHaIM6b0bL7KK4ZOg
exaWuKQusNZ6/h03apzMoN+wTinsT8DQ0j5pPykFxgEb028l8jvmq38GXb+Ui27J
DrDlQ2j5VMjg5YxsFNencq/mKZ9hR39FsnmAvrAYeb308dgaGJhzQSeYQd1bONHw
lkHvWCHbOHPX8WbkHZaemP+/+R6TYUxD85K19w5H8mmSD//IFTAVOqMNm/0/aA1E
N36a1rBcD/9Ie3nQXM38WQgxP/XkqHg5eg78rE+oRp2zrg9Wejwjt5h8aGST6SBA
qS3tyM6i1JW/f4wstWTjJHC2WA+X99QgriuHFAjZSRVlfqBDvt/VA0eKuJsk+iDZ
BB0yJUvXiQQLhmGK2DvwdmJhBiw0tyuHKln1vgqoFrqdLoWZCZK9q8+2RiHxHvvy
lgowkzbF/K8bWKgKBqHKIyfKWoeVa+DV/WCe6bbsyHUhVfOubYGvYHWfIUx4qFaF
OFoSxT6M9V0tfv60CkcQRuhPf5CwLP06BQcP8FmhJCAHZYoUDjEs3Z33n0v0IT/R
gLPoiaRdPJbTKXaTG9fadptiU/Fcnehyxrs89d6vFE0YT8ppMniOfEJ3l4NY8qzu
j8NsylGnX42WMzTrqiy6iZ/RVbGsGrsrhghA8cIiYurFG4Nf3Rei8vWQ7md379wl
7WAPyKvMe/eGKNu9gqrCI90gXtwMDtDnlEnCTC+/UI6fKVTMvTgrQZMCpTmb2zgk
1piLvJTf1uTY3/D3CBusoMI9AjHsHxR8/4ZjZUE8+kvWzAxeVQAD4a3tsGe3cEo2
FbhhQJvDHQnMNN2/2X4B3yggQLkGwtWKVY9Kfmy37n9MoD8oKHL+bnfDXKOVji36
YIyxlDV09WBqcGI8Ryv9SW+HtjU7NfmKBEXyb29J67rvANN02KbobA==
=3cRJ
-----END PGP PUBLIC KEY BLOCK-----
`,
		},
	}

	serviceResult, err := ot.GetService(serviceID)
	if err != nil {
		t.Fatal(err)
	}

	if !assert.Equal(t, service, serviceResult) {
		t.Fatal(err)
	}
}

func TestOnionTree_ListServices(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceIDs := []string{"oniontree"}

	serviceIDsResult, err := ot.ListServices()
	if err != nil {
		t.Fatal(err)
	}

	if !assert.Equal(t, serviceIDs, serviceIDsResult) {
		t.Fatal(err)
	}
}

func TestOnionTree_ListServicesWithTag(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	tag := oniontree.Tag("link_list")
	serviceIDs := []string{"oniontree"}

	serviceIDsResult, err := ot.ListServicesWithTag(tag)
	if err != nil {
		t.Fatal(err)
	}

	if !assert.Equal(t, serviceIDs, serviceIDsResult) {
		t.Fatal(err)
	}
}

func TestOnionTree_ListTags(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	tagsExpected := []oniontree.Tag{"link_list"}

	tagsActual, err := ot.ListTags()
	if err != nil {
		t.Fatal(err)
	}

	if !assert.Equal(t, tagsExpected, tagsActual) {
		t.Fatal(err)
	}
}

func TestOnionTree_ListServiceTags(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceID := "oniontree"
	tagsExpected := []oniontree.Tag{"link_list"}

	tagsActual, err := ot.ListServiceTags(serviceID)
	if err != nil {
		t.Fatal(err)
	}

	if !assert.Equal(t, tagsExpected, tagsActual) {
		t.Fatal(err)
	}
}

func TestOnionTree_TagService(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceID := "oniontree"
	tag := oniontree.Tag("test")

	if err := ot.TagService(serviceID, []oniontree.Tag{tag}); err != nil {
		t.Fatal(err)
	}

	if !assert.FileExists(t, ot.TaggedDir()+"/"+tag.String()+"/oniontree.yaml") {
		t.Fatalf("file '/%s/oniontree.yaml' not exists", tag)
	}
}

func TestOnionTree_TagServiceErrorInvalidTagName(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceID := "oniontree"
	tag := oniontree.Tag("bad_tag")

	err := ot.TagService(serviceID, []oniontree.Tag{tag})
	if _, ok := err.(*oniontree.ErrInvalidTagName); !ok {
		if err == nil {
			t.Fatal("service tagged even though tag name is invalid")
		} else {
			t.Fatal("unexpected error", err.Error())
		}
	}
}

func TestOnionTree_UntagService(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceID := "oniontree"
	tag := oniontree.Tag("link_list")

	if err := ot.UntagService(serviceID, []oniontree.Tag{tag}); err != nil {
		t.Fatal(err)
	}

	if !assert.NoFileExists(t, ot.TaggedDir()+"/"+tag.String()+"/oniontree.yaml") {
		t.Fatalf("file '/%s/oniontree.yaml' exists", tag)
	}
}

func TestOnionTree_VerifySignedMessage(t *testing.T) {
	ot, cleanup := copyOnionTree(t)
	defer cleanup()

	serviceID := "oniontree"
	signedText := `-----BEGIN PGP SIGNED MESSAGE-----
Hash: SHA512

http://onions53ehmf4q75.onion
https://oniontree.org
-----BEGIN PGP SIGNATURE-----

iQIzBAEBCgAdFiEE8B/tR5eVVMktn1ay5LbKxJskKkQFAl3mwUoACgkQ5LbKxJsk
KkS7kw/+KYFiTv7Z0vAxU07tSdEE/w5JGCnhBKHwgoxuM0fa09bknDMyLPLi9nIz
HnJu8+f5+yktbsObX4Hr8jCs8NK9LKBc75uORmlqcilzmPTHQ0suBnURsP8+iPLi
qsDB5kkLzEX1lLfVaSWyIMy8UfXyWeJvDWagQUfP3w6kTS3NvjobIcS5ZyEApzxn
/d9wyEhI1uKp0ai5koLMTHQQu02pIFiykH0n8OiroAjgPZpb1HzQvj/3Ylny4Yey
qRsxSWX0YueGLUMuCrAEjBemooguoEuN8bCjvWpN+rqO0TBWr9KWRFdDw9q42mR7
ju/myQUlKnxNxD4VqhEcczz7BeqxnB60SGd1/IJvNDVEc0aNqt963A81r0DFhOaR
Z1ItUYT4Jpd5xPtHWONmQdVr8Wa45g+XhHmGiTKVAwHA8vQLCOlnZji03ElVq5T+
/Zjs+x2QnUvzut5ohjRpjaoxKk2dhc+D1gAuQ/xzyKT2679zrJkaUdIR0ycijbJ6
togmI1x+j4a8qCPmmJNYGYicf7h618VmGMnWElKfCvBOWne8uIZyWTttivKCiR8j
KFnmLRTsnTsoIJ1lDQ/xqXAPzUIu/TP0Omkjk5+UpofqBZEfzR9tPJFut0MMLXn1
C9eumAqFSLZeMtTdG7LzXo1Iby2MnKjWowvifyhUOh3ohl0bLu8=
=ETay
-----END PGP SIGNATURE-----`

	service, err := ot.GetService(serviceID)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := clearsign.Decode([]byte(signedText))
	if block == nil {
		t.Fatal("invalid clearsigned data")
	}

	_, err = openpgp.CheckDetachedSignature(service.PublicKeys, bytes.NewReader(block.Bytes), block.ArmoredSignature.Body)
	if err != nil {
		t.Fatal(err)
	}
}
