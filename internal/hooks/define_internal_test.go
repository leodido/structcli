package internalhooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnumHelpText_SortsAndFormats(t *testing.T) {
	levels := map[int8][]string{
		2:  {"warn"},
		-1: {"debug"},
		0:  {"info"},
		4:  {"error"},
	}

	values, descr := enumHelpText(levels, "log level")

	assert.Equal(t, []string{"debug", "info", "warn", "error"}, values)
	assert.Equal(t, "log level {debug,info,warn,error}", descr)
}

func TestEnumHelpText_EmptyMap(t *testing.T) {
	levels := map[int][]string{}

	values, descr := enumHelpText(levels, "empty")

	assert.Empty(t, values)
	assert.Equal(t, "empty {}", descr)
}

func TestEnumHelpText_SingleEntry(t *testing.T) {
	levels := map[int][]string{
		0: {"only"},
	}

	values, descr := enumHelpText(levels, "one")

	assert.Equal(t, []string{"only"}, values)
	assert.Equal(t, "one {only}", descr)
}
