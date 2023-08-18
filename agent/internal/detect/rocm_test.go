package detect

import (
	"testing"

	"gotest.tools/assert"
)

const testRocmSmiData = `{
"card0" : {
   "GPU ID" : "0x738c",
   "Unique ID" : "0x7ef570db1b346018",
   "Card SKU" : "D34304",
   "Card vendor" : "0x1002",
   "PCI Bus" : "0000:63:00.0",
   "Card model" : "0x1002"
},
"card3" : {
   "Card model" : "0x1002",
   "Card vendor" : "0x1002",
   "PCI Bus" : "0000:03:00.0",
   "Unique ID" : "0x2f94151ef4b53e39",
   "GPU ID" : "0x738c",
   "Card SKU" : "D34304"
},
"card2" : {
   "Card model" : "0x1002",
   "Card vendor" : "0x1002",
   "PCI Bus" : "0000:26:00.0",
   "Card SKU" : "D34304",
   "GPU ID" : "0x738c",
   "Unique ID" : "0xa8a0e90b7ea1c0ad"
},
"card1" : {
   "GPU ID" : "0x738c",
   "Unique ID" : "0x6be2ee3b2b314cfc",
   "Card SKU" : "D34304",
   "PCI Bus" : "0000:43:00.0",
   "Card vendor" : "0x1002",
   "Card model" : "0x1002"
}
}
`

func TestRocmSmiParser(t *testing.T) {
	testData := []byte(testRocmSmiData)
	result, err := parseRocmSmi(testData)
	assert.NilError(t, err)
	assert.Equal(t, len(result), 4)
	assert.Equal(t, result[1].UUID, "0x6be2ee3b2b314cfc")
}

func Test_deviceAllocated(t *testing.T) {
	type args struct {
		deviceIndex      int
		allocatedDevices []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "VISIBLE_DEVICES not defined",
			args: args{
				deviceIndex:      0,
				allocatedDevices: nil,
			},
			want: true,
		},
		{
			name: "Device allocated",
			args: args{
				deviceIndex:      0,
				allocatedDevices: []string{"0", "1"},
			},
			want: true,
		},
		{
			name: "Device not allocated",
			args: args{
				deviceIndex:      2,
				allocatedDevices: []string{"0", "1"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deviceAllocated(tt.args.deviceIndex, tt.args.allocatedDevices); got != tt.want {
				t.Errorf("deviceAllocated() = %v, want %v", got, tt.want)
			}
		})
	}
}
