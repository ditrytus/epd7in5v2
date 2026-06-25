package epd7in5v2

type Command byte

const (
	CommandPSR    = 0x00 // Panel Setting
	CommandPWR    = 0x01 // Power Setting
	CommandPOF    = 0x02 // Power OFF
	CommandPFS    = 0x03 // Power OFF Sequence Setting
	CommandPON    = 0x04 // Power ON
	CommandPMES   = 0x05 // Power ON Measure
	CommandBTST   = 0x06 // Booster Soft Start
	CommandDSLP   = 0x07 // Deep sleep
	CommandDTM1   = 0x10 // Display Start Transmission 1
	CommandDSP    = 0x11 // Data Stop
	CommandDRF    = 0x12 // Display Refresh
	CommandDTM2   = 0x13 // Display Start Transmission 2
	CommandDUSPI  = 0x15 // Dual SPI
	CommandAUTO   = 0x17 // Auto Sequence
	CommandPLL    = 0x30 // PLL control
	CommandTSC    = 0x40 // Temperature Sensor Calibration
	CommandTSE    = 0x41 // Temperature Sensor Selection
	CommandTSW    = 0x42 // Temperature Sensor Write
	CommandTSR    = 0x43 // Temperature Sensor Read
	CommandPBC    = 0x44 // Panel Break Check
	CommandCDI    = 0x50 // VCOM and data interval setting
	CommandLPD    = 0x51 // Lower Power Detection
	CommandEVS    = 0x52 // End Voltage Settings
	CommandTCON   = 0x60 // TCON setting
	CommandTRES   = 0x61 // Resolution setting
	CommandGSST   = 0x65 // Gate/Source Start setting
	CommandREV    = 0x70 // Revision
	CommandFLG    = 0x71 // Get Status
	CommandAMV    = 0x80 // Auto Measurement VCOM
	CommandVV     = 0x81 // Read VCOM Value
	CommandVDSC   = 0x82 // VCOM_DC Setting
	CommandPTL    = 0x90 // Partial Window
	CommandPTIN   = 0x91 // Partial In
	CommandPTOUT  = 0x92 // Partial Out
	CommandCCSET  = 0xE0 // Cascade Settings
	CommandPWS    = 0xE3 // Power Saving
	CommandLVSEL  = 0xE4 // LVD Voltage Select
	CommandTSSET  = 0xE5 // Force Temperature
	CommandTSDBRY = 0xE7 // Temperature Boundary Phase-C2
)
