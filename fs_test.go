package simba_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/saisona/simba"
)

func TestDownloadFileNoUrl(t *testing.T) {
	dest := fmt.Sprintf("/tmp/%d.dat", time.Now().UnixMilli())
	t.Log(dest)
	err := simba.DownloadFile(dest, "", false)
	if err == nil || err.Error() != "Get \"\": unsupported protocol scheme \"\"" {
		t.Fatalf("got: %s, wanted : nil", err.Error())
	}
}

func TestDownloadFileOverideSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	dest := fmt.Sprintf("%s/%d.dat", tmpDir, time.Now().UnixMilli())
	t.Log(dest)
	err := simba.DownloadFile(dest, "http://www.ovh.net/files/1Mio.dat", true)
	if err != nil {
		t.Fatalf("got: %s, wanted : nil", err.Error())
	}
}

func TestDownloadFileAlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	dest := fmt.Sprintf("%s/%d.dat", tmpDir, time.Now().UnixMilli())
	t.Log(dest)
	err := simba.DownloadFile(dest, "http://www.ovh.net/files/1Mio.dat", true)
	if err != nil {
		t.Fatalf("got: %s, wanted : nil", err.Error())
	}
	err = simba.DownloadFile(dest, "http://www.ovh.net/files/1Mio.dat", false)
	if err != nil {
		t.Fatalf("got: %s, wanted : nil", err.Error())
	}
}

func TestDownloadFileSuccess(t *testing.T) {
	dest := fmt.Sprintf("/tmp/%d.dat", time.Now().UnixMilli())
	t.Log(dest)
	err := simba.DownloadFile(dest, "http://www.ovh.net/files/1Mio.dat", false)
	if err != nil {
		t.Fatalf("got: %s, wanted : nil", err.Error())
	}
}
