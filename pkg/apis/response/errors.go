package response

var errors = map[ErrCode]string{
	ErrCodeMalformedJSON:                            "The JSON you provided was not well-formed or did not validate against our published format.",
	ErrCodeResourceExists:                           "[%s] already exists.",
	ErrCodeResourceNotFound:                         "[%s] not found.",
	ErrCodeTimeZoneInvalid:                          "TimeZone [%s] not valid.",
	ErrCodeEmailServerUnconfigured:                  "Email server not configured.",
	ErrCodeWeComServerUnconfigured:                  "WeCom server not configured.",
	ErrCodeWeChatServerUnconfigured:                 "WeChat server not configured.",
	ErrCodeEmailAddressInvalid:                      "Email addr [%s] not valid.",
	ErrCodeWebhookUriInvalid:                        "Webhook URI [%s] not valid.",
	ErrCodeRecipientNotFound:                        "Recipient [%s] not found.",
	ErrCodeMessageTmplNotFound:                      "Message template [%s] not found.",
	ErrCodeMessageTmplContentInvalid:                "Invalid content [%s].",
	ErrCodeRuleExpressionInvalid:                    "Invalid expression [%s].",
	ErrCodeRuleExpressionOperandInvalid:             "Invalid operand [%s], it should be <propertySetName>\u001f<propertyName>.",
	ErrCodeThingNotFound:                            "Thing [%s] not found.",
	ErrCodePropertyOfThingNotFound:                  "Property %s is not defined in property set %s of thing %s",
	ErrCodeRealTimeRuleRefersMoreThanOnePropertySet: "Real time rule refers more than one property set.",
	ErrCodeVirtualParameterOverlapsOperand:          "Virtual parameter [%s] overlaps operand.",
	ErrCodeVirtualParameterMismatchesProperty:       "Virtual parameter [%s] mismatches property datatype[%s:%s] or unit[%s:%s] or length[%d:%d]",
	ErrCodeTemplateMessageIdNotFound:                "Template [%s] not found.",
	ErrCodeSimpleMessageContentNotFound:             "Simple message content not found.",
	ErrCodeParentTypeNotFound:                       "Parent type [%s] not found.",
	ErrCodePropertySetTypeNotFound:                  "Property set type [%s] not found.",
	ErrCodeThingTypeNotFound:                        "Thing type [%s] not found.",
	ErrCodeThingTypeNotInstantiable:                 "Thing type is not instantiable.",
	ErrCodeAgentTypeNotFound:                        "Agent type [%s] not found.",
	ErrCodeDataSourceNotFound:                       "Data source [%s] not found.",
	ErrCodeDataPointNotFound:                        "Data point [%s] not found.",
	ErrCodePropertySetNotFound:                      "Property set [%s] not found.",
	ErrCodePropertyNotFound:                         "Property [%s] not found.",
	ErrCodeDatatypeMismatch:                         "Data type mismatches.",
	ErrCodeDataUnitMismatch:                         "Data unit mismatches.",
	ErrCodeAccessModeMismatch:                       "Access mode mismatches.",
	ErrCodeDataPointMappingExists:                   "Data point mapping already exists.",
	ErrCodeCommandOptionExists:                      "Command option [%s] already exists.",
	ErrCodeCommandTypeNotFound:                      "Command type [%s] not found.",
	ErrCodeCommandOptionNotFound:                    "Command option [%s] not found.",
	ErrCodeCommandOptionValueMissed:                 "Command option [%s]'s value missed.",
	ErrCodeIntegerInvalid:                           "Integer [%s] not valid. %s",
	ErrCodeLongInvalid:                              "Long [%s] not valid. %s",
	ErrCodeDoubleInvalid:                            "Double [%s] not valid. %s",
	ErrCodeBooleanInvalid:                           "Boolean [%s] not valid. %s",
	ErrCodeTimestampInvalid:                         "Timestamp [%s] not valid. %s",
	ErrCodeStringTooLong:                            "String [%s] not valid. %s",
	ErrCodeEnumInvalid:                              "Enum [%s] not valid. %v",
	ErrCodeMapInvalid:                               "Map [%s] not valid. %v",
	ErrCodeLinkInvalid:                              "Link [%s] not valid.",
	ErrCodePropertyNotWritable:                      "Property [%s] not writable.",
	ErrCodeLegalActionNotFound:                      "Legal action not found.",
	ErrCodeDeliverFailed:                            "Deliver command/action failed.",
	ErrCodeDataPointMappingNotFound:                 "Data point mapping not found.",
	ErrCodeResourceDeleting:                         "Resource [%s] deleting.",
	ErrCodeAssociatedResourceDeletionFailed:         "Failed deleting the associated resource of [%s].",
	ErrCodeTimeInvalid:                              "Time [%s] invalid.",
	ErrCodeRootThingTypeNotBaseAgent:                "Root thing type not BaseAgent.",
	ErrCodeImageNameNotFound:                        "name is required.",
	ErrCodeImageFileNotFound:                        "file is required.",
	ErrCodeDuplicatedComponent:                      "Component [%s] duplicated",
	ErrCodeComponentUriInvalid:                      "URI [%s] of component [%s] not valid.",
	ErrCodeChildTypeExist:                           "Resource [%s] has child type.",
	ErrCodeAssociateResourceExist:                   "Resource [%s] is referred.",
	ErrCodeAllSelectInvalid:                         "All selects not valid.",
	ErrCodeRollupIntervalInvalid:                    "Interval [%s] not valid.",
	ErrCodeRollupStartInvalid:                       "Rollup start time [%s] not valid.",
	ErrCodeRollupEndInvalid:                         "Rollup end time [%s] not valid.",
	ErrCodeDuplicatedField:                          "Field [%s] duplicated.",
	ErrCodeFieldExists:                              "Field [%s] already exists in [%s].",
	ErrCodeRequiredFieldMissed:                      "Required field [%s] missed.",
	ErrCodeFieldInvalid:                             "The value %v of the field [%s] not valid.",
	ErrCodeFieldUnSupported:                         "Unsupported datatype %s.",
	ErrCodeDeleteFundamentalResourceFailed:          "Resource [%s] is fundamental and cannot be deleted.",
	ErrCodeInvalidValue:                             `%s`,
	ErrCodeInvalidUpdate:                            `%s`,
	ErrCodeTooManyJsonPatchOperations:               `Json Patch operations exceeds %d.`,
	ErrCodeCharacteristicNotFound:                   "Characteristic [%s] not found.",
	ErrCodeCycledThing:                              "Cycled thing [%s].",
}

// !!! IMPORTANT PLEASE READ FIRST !!!
// You SHOULD add new code at the end of enum firstly.

var ErrMalformedJSON = &responseError{
	Code:    ErrCodeMalformedJSON,
	Message: errors[ErrCodeMalformedJSON],
}

var ErrEmailServerUnconfigured = &responseError{
	Code:    ErrCodeEmailServerUnconfigured,
	Message: errors[ErrCodeEmailServerUnconfigured],
}

var ErrWeComServerUnconfigured = &responseError{
	Code:    ErrCodeWeComServerUnconfigured,
	Message: errors[ErrCodeWeComServerUnconfigured],
}

var ErrWeChatServerUnconfigured = &responseError{
	Code:    ErrCodeWeChatServerUnconfigured,
	Message: errors[ErrCodeWeChatServerUnconfigured],
}

var ErrRealTimeRuleRefersMoreThanOnePropertySet = &responseError{
	Code:    ErrCodeRealTimeRuleRefersMoreThanOnePropertySet,
	Message: errors[ErrCodeRealTimeRuleRefersMoreThanOnePropertySet],
}

var ErrSimpleMessageContentNotFound = &responseError{
	Code:    ErrCodeSimpleMessageContentNotFound,
	Message: errors[ErrCodeSimpleMessageContentNotFound],
}

var ErrThingTypeNotInstantiable = &responseError{
	Code:    ErrCodeThingTypeNotInstantiable,
	Message: errors[ErrCodeThingTypeNotInstantiable],
}

var ErrDatatypeMismatch = &responseError{
	Code:    ErrCodeDatatypeMismatch,
	Message: errors[ErrCodeDatatypeMismatch],
}

var ErrDataUnitMismatch = &responseError{
	Code:    ErrCodeDataUnitMismatch,
	Message: errors[ErrCodeDataUnitMismatch],
}

var ErrAccessModeMismatch = &responseError{
	Code:    ErrCodeAccessModeMismatch,
	Message: errors[ErrCodeAccessModeMismatch],
}

var ErrDataPointMappingExists = &responseError{
	Code:    ErrCodeDataPointMappingExists,
	Message: errors[ErrCodeDataPointMappingExists],
}

var ErrLegalActionNotFound = &responseError{
	Code:    ErrCodeLegalActionNotFound,
	Message: errors[ErrCodeLegalActionNotFound],
}

var ErrDeliverFailed = &responseError{
	Code:    ErrCodeDeliverFailed,
	Message: errors[ErrCodeDeliverFailed],
}

var ErrDataPointMappingNotFound = &responseError{
	Code:    ErrCodeDataPointMappingNotFound,
	Message: errors[ErrCodeDataPointMappingNotFound],
}

var ErrRootThingTypeNotBaseAgent = &responseError{
	Code:    ErrCodeRootThingTypeNotBaseAgent,
	Message: errors[ErrCodeRootThingTypeNotBaseAgent],
}

var ErrImageNameNotFound = &responseError{
	Code:    ErrCodeImageNameNotFound,
	Message: errors[ErrCodeImageNameNotFound],
}

var ErrImageFileNotFound = &responseError{
	Code:    ErrCodeImageFileNotFound,
	Message: errors[ErrCodeImageFileNotFound],
}

var ErrAllSelectInvalid = &responseError{
	Code:    ErrCodeAllSelectInvalid,
	Message: errors[ErrCodeAllSelectInvalid],
}
