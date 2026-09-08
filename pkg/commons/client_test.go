package commons

import "testing"

func TestParseClientType(t *testing.T) {
	testCases := []struct {
		name    string
		configs map[string]string
		want    ClientType
		wantErr bool
	}{
		{name: "default", configs: map[string]string{}, want: IrodsFuseClientType},
		{name: "irods", configs: map[string]string{"client": "iRoDsFuSe"}, want: IrodsFuseClientType},
		{name: "webdav", configs: map[string]string{"client": "webdav"}, want: WebdavClientType},
		{name: "nfs", configs: map[string]string{"client": "nfs"}, want: NfsClientType},
		{name: "unknown", configs: map[string]string{"client": "typo"}, wantErr: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ParseClientType(testCase.configs)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("ParseClientType() error = %v, wantErr %t", err, testCase.wantErr)
			}
			if got != testCase.want {
				t.Errorf("ParseClientType() = %q, want %q", got, testCase.want)
			}
		})
	}
}
