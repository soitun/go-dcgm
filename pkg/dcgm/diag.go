package dcgm

/*
#include "dcgm_agent.h"
#include "dcgm_structs.h"
*/
import "C"

import (
	"unsafe"
)

const DIAG_RESULT_STRING_SIZE = 1024

type DiagType int

const (
	DiagQuick    DiagType = 1
	DiagMedium            = 2
	DiagLong              = 3
	DiagExtended          = 4
)

type DiagResult struct {
	Status       string
	TestName     string
	TestOutput   string
	ErrorCode    uint
	ErrorMessage string
}

type DiagResults struct {
	Software []DiagResult
}

func diagResultString(r int) string {
	switch r {
	case C.DCGM_DIAG_RESULT_PASS:
		return "pass"
	case C.DCGM_DIAG_RESULT_SKIP:
		return "skipped"
	case C.DCGM_DIAG_RESULT_WARN:
		return "warn"
	case C.DCGM_DIAG_RESULT_FAIL:
		return "fail"
	case C.DCGM_DIAG_RESULT_NOT_RUN:
		return "notrun"
	}
	return ""
}

func swTestName(t int) string {
	switch t {
	case C.DCGM_SWTEST_DENYLIST:
		return "presence of drivers on the denylist (e.g. nouveau)"
	case C.DCGM_SWTEST_NVML_LIBRARY:
		return "presence (and version) of NVML lib"
	case C.DCGM_SWTEST_CUDA_MAIN_LIBRARY:
		return "presence (and version) of CUDA lib"
	case C.DCGM_SWTEST_CUDA_RUNTIME_LIBRARY:
		return "presence (and version) of CUDA RT lib"
	case C.DCGM_SWTEST_PERMISSIONS:
		return "character device permissions"
	case C.DCGM_SWTEST_PERSISTENCE_MODE:
		return "persistence mode enabled"
	case C.DCGM_SWTEST_ENVIRONMENT:
		return "CUDA environment vars that may slow tests"
	case C.DCGM_SWTEST_PAGE_RETIREMENT:
		return "pending frame buffer page retirement"
	case C.DCGM_SWTEST_GRAPHICS_PROCESSES:
		return "graphics processes running"
	case C.DCGM_SWTEST_INFOROM:
		return "inforom corruption"
	}

	return ""
}

func getErrorMsg(entityId uint, response C.dcgmDiagResponse_v11) (string, uint) {
	for i := 0; i < int(response.numErrors); i++ {
		if uint(response.errors[i].entity.entityId) != entityId {
			continue
		}

		msg := C.GoString((*C.char)(unsafe.Pointer(&response.errors[i].msg)))
		code := uint(response.errors[i].code)
		return msg, code
	}

	return "", 0
}

func getInfoMsg(entityId uint, response C.dcgmDiagResponse_v11) string {
	for i := 0; i < int(response.numInfo); i++ {
		if uint(response.info[i].entity.entityId) != entityId {
			continue
		}

		msg := C.GoString((*C.char)(unsafe.Pointer(&response.info[i].msg)))
		return msg
	}

	return ""
}

func newDiagResult(resultIndex uint, response C.dcgmDiagResponse_v11) DiagResult {
	entityId := uint(response.results[resultIndex].entity.entityId)

	msg, code := getErrorMsg(entityId, response)
	info := getInfoMsg(entityId, response)
	testName := swTestName(int(response.results[resultIndex].testId))

	return DiagResult{
		Status:       diagResultString(int(response.results[resultIndex].result)),
		TestName:     testName,
		TestOutput:   info,
		ErrorCode:    uint(code),
		ErrorMessage: msg,
	}
}

func diagLevel(diagType DiagType) C.dcgmDiagnosticLevel_t {
	switch diagType {
	case DiagQuick:
		return C.DCGM_DIAG_LVL_SHORT
	case DiagMedium:
		return C.DCGM_DIAG_LVL_MED
	case DiagLong:
		return C.DCGM_DIAG_LVL_LONG
	case DiagExtended:
		return C.DCGM_DIAG_LVL_XLONG
	}
	return C.DCGM_DIAG_LVL_INVALID
}

func RunDiag(diagType DiagType, groupId GroupHandle) (DiagResults, error) {
	var diagResults C.dcgmDiagResponse_v11
	diagResults.version = makeVersion11(unsafe.Sizeof(diagResults))

	result := C.dcgmRunDiagnostic(handle.handle, groupId.handle, diagLevel(diagType), (*C.dcgmDiagResponse_v11)(unsafe.Pointer(&diagResults)))
	if err := errorString(result); err != nil {
		return DiagResults{}, &DcgmError{msg: C.GoString(C.errorString(result)), Code: result}
	}

	var diagRun DiagResults
	for i := 0; i < int(diagResults.numResults); i++ {
		dr := newDiagResult(uint(diagResults.results[i].entity.entityId), diagResults)
		diagRun.Software = append(diagRun.Software, dr)
	}

	return diagRun, nil
}
