package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

var (
	hidDLL                          = syscall.NewLazyDLL("hid.dll")
	setupapi                        = syscall.NewLazyDLL("setupapi.dll")
	kernel32                        = syscall.NewLazyDLL("kernel32.dll")
	hidD_GetHidGuid                 = hidDLL.NewProc("HidD_GetHidGuid")
	hidD_GetAttributes              = hidDLL.NewProc("HidD_GetAttributes")
	hidD_SetFeature                 = hidDLL.NewProc("HidD_SetFeature")
	setupDiGetClassDevs             = setupapi.NewProc("SetupDiGetClassDevsW")
	setupDiEnumDeviceInterfaces     = setupapi.NewProc("SetupDiEnumDeviceInterfaces")
	setupDiGetDeviceInterfaceDetail = setupapi.NewProc("SetupDiGetDeviceInterfaceDetailW")
	setupDiDestroyDeviceInfoList    = setupapi.NewProc("SetupDiDestroyDeviceInfoList")
	createFile                      = kernel32.NewProc("CreateFileW")
	closeHandle                     = kernel32.NewProc("CloseHandle")
	writeFile                       = kernel32.NewProc("WriteFile")
)

const (
	DIGCF_PRESENT         = 0x00000002
	DIGCF_DEVICEINTERFACE = 0x00000010
	INVALID_HANDLE_VALUE  = ^uintptr(0)
	GENERIC_READ          = 0x80000000
	GENERIC_WRITE         = 0x40000000
	FILE_SHARE_READ       = 0x00000001
	FILE_SHARE_WRITE      = 0x00000002
	OPEN_EXISTING         = 3
	ERROR_NO_MORE_ITEMS   = 259
)

type GUID struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type SP_DEVICE_INTERFACE_DATA struct {
	cbSize             uint32
	InterfaceClassGuid GUID
	Flags              uint32
	Reserved           uintptr
}

type SP_DEVICE_INTERFACE_DETAIL_DATA struct {
	cbSize     uint32
	DevicePath [1]uint16
}

type HIDD_ATTRIBUTES struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

type WindowsMouseController struct {
	handle     syscall.Handle
	devicePath string
	attributes HIDD_ATTRIBUTES
}

func NewWindowsMouseController() (*WindowsMouseController, error) {
	devicePath, attributes, err := findLAMZUDeviceWindows()
	if err != nil {
		return nil, fmt.Errorf("failed to find LAMZU device: %w", err)
	}

	handle, err := openDeviceHandle(devicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open device: %w", err)
	}

	return &WindowsMouseController{
		handle:     handle,
		devicePath: devicePath,
		attributes: attributes,
	}, nil
}

func (w *WindowsMouseController) Close() {
	if w.handle != syscall.InvalidHandle {
		closeHandle.Call(uintptr(w.handle))
		w.handle = syscall.InvalidHandle
	}
}

func (w *WindowsMouseController) TestConnection() error {
	if w.handle == syscall.InvalidHandle {
		return fmt.Errorf("device not connected")
	}

	if verbose {
		fmt.Printf("🔌 Connected to LAMZU device via Windows API: VID=0x%04X, PID=0x%04X\n",
			w.attributes.VendorID, w.attributes.ProductID)
	}

	return nil
}

func (w *WindowsMouseController) SetPollingRate(rate int) error {
	rateValue, exists := pollingRateMap[rate]
	if !exists {
		return fmt.Errorf("invalid polling rate: %d", rate)
	}

	// Use exact format from working TypeScript implementation
	command := make([]byte, REPORT_SIZE)
	command[0] = 0x00      // Report ID
	command[1] = 0x00      // Padding (default fill)
	command[2] = 0x00      // Padding (default fill)
	command[3] = 0x02      // Command type
	command[4] = 0x02      // Sub-command
	command[5] = 0x01      // Parameter
	command[6] = 0x00      // Reserved
	command[7] = 1         // Configuration
	command[8] = rateValue // Polling rate value

	if verbose {
		fmt.Printf("🔧 Sending command: [%02X %02X %02X %02X %02X %02X %02X %02X %02X...]\n",
			command[0], command[1], command[2], command[3], command[4], command[5], command[6], command[7], command[8])
	}

	// Try HidD_SetFeature first (for feature reports)
	ret, _, err := hidD_SetFeature.Call(
		uintptr(w.handle),
		uintptr(unsafe.Pointer(&command[0])),
		uintptr(len(command)),
	)

	if ret != 0 {
		if verbose {
			fmt.Printf("📡 Polling rate set to %dHz (value: %d) via HidD_SetFeature\n", rate, rateValue)
		}
		return nil
	}

	// If HidD_SetFeature fails, try WriteFile (for output reports)
	if verbose {
		fmt.Printf("⚠️ HidD_SetFeature failed (%v), trying WriteFile...\n", err)
	}

	var bytesWritten uint32
	ret, _, err = writeFile.Call(
		uintptr(w.handle),
		uintptr(unsafe.Pointer(&command[0])),
		uintptr(len(command)),
		uintptr(unsafe.Pointer(&bytesWritten)),
		0,
	)

	if ret == 0 {
		return fmt.Errorf("failed to write command (both HidD_SetFeature and WriteFile failed): %v", err)
	}

	if verbose {
		fmt.Printf("📡 Polling rate set to %dHz (value: %d) via WriteFile (%d bytes written)\n", rate, rateValue, bytesWritten)
	}

	return nil
}

func (w *WindowsMouseController) GetDeviceInfo() (*HIDD_ATTRIBUTES, error) {
	return &w.attributes, nil
}

func findLAMZUDeviceWindows() (string, HIDD_ATTRIBUTES, error) {
	var hidGuid GUID

	hidD_GetHidGuid.Call(uintptr(unsafe.Pointer(&hidGuid)))

	if verbose {
		fmt.Printf("🔍 HID GUID: {%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X}\n",
			hidGuid.Data1, hidGuid.Data2, hidGuid.Data3,
			hidGuid.Data4[0], hidGuid.Data4[1], hidGuid.Data4[2], hidGuid.Data4[3],
			hidGuid.Data4[4], hidGuid.Data4[5], hidGuid.Data4[6], hidGuid.Data4[7])
	}

	hDevInfo, _, _ := setupDiGetClassDevs.Call(
		uintptr(unsafe.Pointer(&hidGuid)),
		0,
		0,
		DIGCF_PRESENT|DIGCF_DEVICEINTERFACE,
	)

	if hDevInfo == INVALID_HANDLE_VALUE {
		return "", HIDD_ATTRIBUTES{}, errors.New("failed to get device list")
	}
	defer setupDiDestroyDeviceInfoList.Call(hDevInfo)

	var deviceIndex uint32 = 0
	var deviceFound bool
	var foundPath string
	var foundAttributes HIDD_ATTRIBUTES

	for {
		var deviceInterfaceData SP_DEVICE_INTERFACE_DATA
		deviceInterfaceData.cbSize = uint32(unsafe.Sizeof(deviceInterfaceData))

		ret, _, _ := setupDiEnumDeviceInterfaces.Call(
			hDevInfo,
			0,
			uintptr(unsafe.Pointer(&hidGuid)),
			uintptr(deviceIndex),
			uintptr(unsafe.Pointer(&deviceInterfaceData)),
		)

		if ret == 0 {
			break
		}

		var requiredSize uint32
		setupDiGetDeviceInterfaceDetail.Call(
			hDevInfo,
			uintptr(unsafe.Pointer(&deviceInterfaceData)),
			0,
			0,
			uintptr(unsafe.Pointer(&requiredSize)),
			0,
		)

		detailData := make([]byte, requiredSize)
		detail := (*SP_DEVICE_INTERFACE_DETAIL_DATA)(unsafe.Pointer(&detailData[0]))
		detail.cbSize = uint32(unsafe.Sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA{}))

		ret, _, _ = setupDiGetDeviceInterfaceDetail.Call(
			hDevInfo,
			uintptr(unsafe.Pointer(&deviceInterfaceData)),
			uintptr(unsafe.Pointer(detail)),
			uintptr(requiredSize),
			0,
			0,
		)

		if ret == 0 {
			deviceIndex++
			continue
		}

		devicePath := syscall.UTF16ToString((*[256]uint16)(unsafe.Pointer(&detail.DevicePath[0]))[:256])

		if verbose {
			fmt.Printf("🔍 Checking device: %s\n", devicePath)
		}

		handle, err := openDeviceHandle(devicePath)
		if err != nil {
			deviceIndex++
			continue
		}

		var attributes HIDD_ATTRIBUTES
		attributes.Size = uint32(unsafe.Sizeof(attributes))

		ret, _, _ = hidD_GetAttributes.Call(
			uintptr(handle),
			uintptr(unsafe.Pointer(&attributes)),
		)

		closeHandle.Call(uintptr(handle))

		if ret != 0 {
			if verbose {
				fmt.Printf("📊 Device VID=0x%04X, PID=0x%04X\n", attributes.VendorID, attributes.ProductID)
			}

			if attributes.VendorID == LAMZU_VID && attributes.ProductID == LAMZU_PID {
				// Extract interface number from device path (mi_XX)
				interfaceNum := extractInterfaceNumber(devicePath)

				if verbose {
					fmt.Printf("🔍 Found LAMZU device interface %d: %s\n", interfaceNum, devicePath)
				}

				// Only use interface 2 (same as karalabe/hid implementation)
				if interfaceNum == INTERFACE_NUMBER {
					if verbose {
						fmt.Printf("✅ Found LAMZU device on correct interface %d: %s\n", interfaceNum, devicePath)
					}
					deviceFound = true
					foundPath = devicePath
					foundAttributes = attributes
					break
				} else if verbose {
					fmt.Printf("⚠️ Skipping LAMZU device on interface %d (need interface %d)\n", interfaceNum, INTERFACE_NUMBER)
				}
			}
		}

		deviceIndex++
	}

	if !deviceFound {
		return "", HIDD_ATTRIBUTES{}, fmt.Errorf("LAMZU device not found - make sure it's connected and you're running as administrator")
	}

	return foundPath, foundAttributes, nil
}

func extractInterfaceNumber(devicePath string) int {
	// Parse device path to extract interface number
	// Example: \\?\hid#vid_373e&pid_001e&mi_02&col01#...
	// We want to extract "02" from "mi_02"

	// Look for "mi_" pattern
	miIndex := strings.Index(strings.ToLower(devicePath), "&mi_")
	if miIndex == -1 {
		return -1 // No interface number found
	}

	// Extract the two digits after "mi_"
	start := miIndex + 4 // Skip "&mi_"
	if start+2 > len(devicePath) {
		return -1
	}

	interfaceStr := devicePath[start : start+2]

	// Convert hex string to int
	if interfaceNum, err := strconv.ParseInt(interfaceStr, 16, 32); err == nil {
		return int(interfaceNum)
	}

	return -1
}

func openDeviceHandle(devicePath string) (syscall.Handle, error) {
	pathPtr, err := syscall.UTF16PtrFromString(devicePath)
	if err != nil {
		return syscall.InvalidHandle, err
	}

	handle, _, _ := createFile.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		GENERIC_READ|GENERIC_WRITE,
		FILE_SHARE_READ|FILE_SHARE_WRITE,
		0,
		OPEN_EXISTING,
		0,
		0,
	)

	if handle == INVALID_HANDLE_VALUE {
		return syscall.InvalidHandle, errors.New("failed to open device")
	}

	return syscall.Handle(handle), nil
}
