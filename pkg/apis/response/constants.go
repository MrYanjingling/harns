package response

type ErrCode int

const (
	_                                               ErrCode = 10000 + iota
	ErrCodeMalformedJSON                                    // 10001
	ErrCodeResourceExists                                   // 10002
	ErrCodeResourceNotFound                                 // 10003
	ErrCodeTimeZoneInvalid                                  // 10004
	ErrCodeEmailServerUnconfigured                          // 10005
	ErrCodeWeComServerUnconfigured                          // 10006
	ErrCodeWeChatServerUnconfigured                         // 10007
	ErrCodeEmailAddressInvalid                              // 10008
	ErrCodeWebhookUriInvalid                                // 10009
	ErrCodeRecipientNotFound                                // 10010
	ErrCodeMessageTmplNotFound                              // 10011
	ErrCodeMessageTmplContentInvalid                        // 10012
	ErrCodeRuleExpressionInvalid                            // 10013
	ErrCodeRuleExpressionOperandInvalid                     // 10014
	ErrCodeThingNotFound                                    // 10015
	ErrCodePropertyOfThingNotFound                          // 10016
	ErrCodeRealTimeRuleRefersMoreThanOnePropertySet         // 10017
	ErrCodeVirtualParameterOverlapsOperand                  // 10018
	ErrCodeVirtualParameterMismatchesProperty               // 10019
	ErrCodeTemplateMessageIdNotFound                        // 10020
	ErrCodeSimpleMessageContentNotFound                     // 10021
	ErrCodeParentTypeNotFound                               // 10022
	ErrCodePropertySetTypeNotFound                          // 10023
	ErrCodeThingTypeNotFound                                // 10024
	ErrCodeThingTypeNotInstantiable                         // 10025
	ErrCodeAgentTypeNotFound                                // 10026
	ErrCodeAgentNotFound                                    // 10027
	ErrCodeDataSourceNotFound                               // 10028
	ErrCodeDataPointNotFound                                // 10029
	ErrCodePropertySetNotFound                              // 10030
	ErrCodePropertyNotFound                                 // 10031
	ErrCodeDatatypeMismatch                                 // 10032
	ErrCodeDataUnitMismatch                                 // 10033
	ErrCodeAccessModeMismatch                               // 10034
	ErrCodeDataPointMappingExists                           // 10035
	ErrCodeCommandOptionExists                              // 10036
	ErrCodeCommandTypeNotFound                              // 10037
	ErrCodeCommandOptionNotFound                            // 10038
	ErrCodeCommandOptionValueMissed                         // 10039
	ErrCodeIntegerInvalid                                   // 10040
	ErrCodeLongInvalid                                      // 10041
	ErrCodeDoubleInvalid                                    // 10042
	ErrCodeBooleanInvalid                                   // 10043
	ErrCodeTimestampInvalid                                 // 10044
	ErrCodeStringTooLong                                    // 10045
	ErrCodeEnumInvalid                                      // 10046
	ErrCodeMapInvalid                                       // 10047
	ErrCodeLinkInvalid                                      // 10048
	ErrCodePropertyNotWritable                              // 10049
	ErrCodeLegalActionNotFound                              // 10050
	ErrCodeDeliverFailed                                    // 10051
	ErrCodeDataPointMappingNotFound                         // 10052
	ErrCodeResourceDeleting                                 // 10053
	ErrCodeAssociatedResourceDeletionFailed                 // 10054
	ErrCodeTimeInvalid                                      // 10055
	ErrCodeRootThingTypeNotBaseAgent                        // 10056
	ErrCodeImageNameNotFound                                // 10057
	ErrCodeImageFileNotFound                                // 10058
	ErrCodeDuplicatedComponent                              // 10059
	ErrCodeComponentUriInvalid                              // 10060
	ErrCodeChildTypeExist                                   // 10061
	ErrCodeAssociateResourceExist                           // 10062
	ErrCodeAllSelectInvalid                                 // 10063
	ErrCodeRollupIntervalInvalid                            // 10064
	ErrCodeRollupStartInvalid                               // 10065
	ErrCodeRollupEndInvalid                                 // 10066
	ErrCodeDuplicatedField                                  // 10067
	ErrCodeFieldExists                                      // 10068
	ErrCodeRequiredFieldMissed                              // 10069
	ErrCodeFieldInvalid                                     // 10070
	ErrCodeFieldUnSupported                                 // 10071
	ErrCodeDeleteFundamentalResourceFailed                  // 10072
	ErrCodeInvalidValue                                     // 10073
	ErrCodeInvalidUpdate                                    // 10074
	ErrCodeTooManyJsonPatchOperations                       // 10075
	ErrCodeCharacteristicNotFound                           // 10076
	ErrCodeCycledThing                                      // 10077
)

// !!! IMPORTANT PLEASE READ FIRST !!!
// You SHOULD add new code at the end, and append comment of number
// Meanwhile, the corresponding error message SHOULD be appended in response.errors
// The order MUST be consistent between them
