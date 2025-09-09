package manager

import "github.com/rackov/NavControl/proto"

func StrToServiceManagerType(s string) (logLevel proto.ServiceManagerType) {
	switch s {
	case "receiver":
		logLevel = proto.ServiceManagerType_RECEIVER
	case "writer":
		logLevel = proto.ServiceManagerType_WRITER
	case "retranslator":
		logLevel = proto.ServiceManagerType_RETRANSLATOR
	default:
		logLevel = proto.ServiceManagerType_RECEIVER // значение по умолчанию
	}
	return
}
