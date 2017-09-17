package goodhosts

import (
	"fmt"
	"reflect"
	"testing"
)

func TestHostsLineIsComment(t *testing.T) {
	comment := "   # This is a comment   "
	line := NewHostsLine(comment)
	result := line.IsComment()
	if !result {
		t.Error(fmt.Sprintf("'%s' should be a comment"), comment)
	}
}

func TestNewHostsLineWithEmptyLine(t *testing.T) {
	line := NewHostsLine("")
	if line.Raw != "" {
		t.Error("Failed to load empty line.")
	}
}

func TestHostsHas(t *testing.T) {
	hosts := new(Hosts)
	hosts.Lines = []HostsLine{
		NewHostsLine("127.0.0.1 yadda"), NewHostsLine("10.0.0.7 nada")}

	// We should find this entry.
	if !hosts.Has("10.0.0.7", "nada") {
		t.Error("Couldn't find entry in hosts file.")
	}

	// We shouldn't find this entry
	if hosts.Has("10.0.0.7", "shuda") {
		t.Error("Found entry that isn't in hosts file.")
	}
}

func TestHostsHasDoesntFindMissingEntry(t *testing.T) {
	hosts := new(Hosts)
	hosts.Lines = []HostsLine{
		NewHostsLine("127.0.0.1 yadda"), NewHostsLine("10.0.0.7 nada")}

	if hosts.Has("10.0.0.7", "brada") {
		t.Error("Found missing entry.")
	}
}

func TestHostsAddWhenIpHasOtherHosts(t *testing.T) {
	hosts := new(Hosts)
	hosts.Lines = []HostsLine{
		NewHostsLine("127.0.0.1 yadda"), NewHostsLine("10.0.0.7 nada yadda")}

	hosts.Add("10.0.0.7", "brada", "yadda")

	expectedLines := []HostsLine{
		NewHostsLine("127.0.0.1 yadda"), NewHostsLine("10.0.0.7 nada yadda brada")}

	if !reflect.DeepEqual(hosts.Lines, expectedLines) {
		t.Error("Add entry failed to append entry.")
	}
}

func TestHostsAddWhenIpDoesntExist(t *testing.T) {
	hosts := new(Hosts)
	hosts.Lines = []HostsLine{
		NewHostsLine("127.0.0.1 yadda")}

	hosts.Add("10.0.0.7", "brada", "yadda")

	expectedLines := []HostsLine{
		NewHostsLine("127.0.0.1 yadda"), NewHostsLine("10.0.0.7 brada yadda")}

	if !reflect.DeepEqual(hosts.Lines, expectedLines) {
		t.Error("Add entry failed to append entry.")
	}
}

func TestHostsRemoveWhenLastHostIpCombo(t *testing.T) {
	hosts := new(Hosts)
	hosts.Lines = []HostsLine{
		NewHostsLine("127.0.0.1 yadda"), NewHostsLine("10.0.0.7 nada")}

	hosts.Remove("10.0.0.7", "nada")

	expectedLines := []HostsLine{NewHostsLine("127.0.0.1 yadda")}

	if !reflect.DeepEqual(hosts.Lines, expectedLines) {
		t.Error("Remove entry failed to remove entry.")
	}
}

func TestHostsRemoveWhenIpHasOtherHosts(t *testing.T) {
	hosts := new(Hosts)

	hosts.Lines = []HostsLine{
		NewHostsLine("127.0.0.1 yadda"), NewHostsLine("10.0.0.7 nada brada")}

	hosts.Remove("10.0.0.7", "nada")

	expectedLines := []HostsLine{
		NewHostsLine("127.0.0.1 yadda"), NewHostsLine("10.0.0.7 brada")}

	if !reflect.DeepEqual(hosts.Lines, expectedLines) {
		t.Error("Remove entry failed to remove entry.")
	}
}

func TestHostsRemoveMultipleEntries(t *testing.T) {
	hosts := new(Hosts)
	hosts.Lines = []HostsLine{
		NewHostsLine("127.0.0.1 yadda nadda prada")}

	hosts.Remove("127.0.0.1", "yadda", "prada")
	if hosts.Lines[0].Raw != "127.0.0.1 nadda" {
		t.Error("Failed to remove multiple entries.")
	}
}
