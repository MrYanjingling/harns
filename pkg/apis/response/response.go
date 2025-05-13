package response

import (
	"encoding/json"
	"fmt"
	v1 "lightiot/pkg/model/v1"
	"strconv"
	"strings"
	"sync"
)

type responseError struct {
	Code    ErrCode `json:"code"`
	Message string  `json:"message"`
	Err     error   `json:"-"`
}

func (re *responseError) Error() string {
	if re == nil {
		return ""
	}
	s := `{
    "code": ` + strconv.Itoa(int(re.Code)) + `,
    "message": ` + re.Message + `
}`
	return s
}

func (re *responseError) GetCode() ErrCode {
	if re == nil {
		return 0
	}
	return re.Code
}

func (re *responseError) Unwrap() error {
	return re.Err
}

func IsResponseError(err error) bool {
	_, ok := err.(*responseError)
	return ok
}

// MultiError contains multiple errors and implements the error interface. Its
// zero value is ready to use. All its methods are goroutine safe.
type MultiError struct {
	mtx    sync.Mutex
	errors []error
}

func NewMultiError(err ...error) *MultiError {
	return &MultiError{
		errors: err,
	}
}

// Add adds an error to the MultiError.
func (e *MultiError) Add(err ...error) {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	e.errors = append(e.errors, err...)
}

// Len returns the number of errors added to the MultiError.
func (e *MultiError) Len() int {
	if e == nil {
		return 0
	}
	e.mtx.Lock()
	defer e.mtx.Unlock()

	return len(e.errors)
}

// MultiError returns the errors added to the MuliError. The returned slice is a
// copy of the internal slice of errors.
func (e *MultiError) Errors() []error {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	return append(make([]error, 0, len(e.errors)), e.errors...)
}

func (e *MultiError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Errors []error `json:"errors"`
	}{
		Errors: e.errors,
	})
}

func (e *MultiError) UnmarshalJSON(bytes []byte) error {
	errs := struct {
		Errors []*responseError `json:"errors"`
	}{}
	if err := json.Unmarshal(bytes, &errs); err != nil {
		return err
	}
	for _, err := range errs.Errors {
		e.Add(err)
	}
	return nil
}

func (e *MultiError) Error() string {
	e.mtx.Lock()
	defer e.mtx.Unlock()

	es := make([]string, 0, len(e.errors))
	for _, err := range e.errors {
		es = append(es, err.Error())
	}
	return strings.Join(es, "; ")
}

func generateError(code ErrCode, s ...interface{}) *responseError {
	return &responseError{
		Code:    code,
		Message: fmt.Sprintf(errors[code], s...),
	}
}

func generateErrorWrapper(code ErrCode, err error, s ...interface{}) *responseError {
	return &responseError{
		Code:    code,
		Message: fmt.Sprintf(errors[code], s...),
		Err:     err,
	}
}

// https://golang.org/doc/faq#convert_slice_of_interface
func convert(infos []string) []interface{} {
	s := make([]interface{}, len(infos))
	for i, v := range infos {
		s[i] = v
	}
	return s
}

func ErrResourceExists(resource string) *responseError {
	return generateError(ErrCodeResourceExists, resource)
}

func ErrResourceNotFound(resource string) *responseError {
	return generateError(ErrCodeResourceNotFound, resource)
}

func ErrTimeZoneInvalid(timeZone string) *responseError {
	return generateError(ErrCodeTimeZoneInvalid, timeZone)
}

func ErrEmailAddressInvalid(addr string) *responseError {
	return generateError(ErrCodeEmailAddressInvalid, addr)
}

func ErrWebhookUriInvalid(addr string) *responseError {
	return generateError(ErrCodeWebhookUriInvalid, addr)
}

func ErrRecipientNotFound(r []string) *responseError {
	return generateError(ErrCodeRecipientNotFound, r)
}

func ErrMessageTmplNotFound(mt []string) *responseError {
	return generateError(ErrCodeMessageTmplNotFound, mt)
}

func ErrMessageTmplContentInvalid(content string) *responseError {
	return generateError(ErrCodeMessageTmplContentInvalid, content)
}

func ErrRuleExpressionInvalid(err string) *responseError {
	return generateError(ErrCodeRuleExpressionInvalid, err)
}

func ErrRuleExpressionOperandInvalid(operand string) *responseError {
	return generateError(ErrCodeRuleExpressionOperandInvalid, operand)
}

func ErrThingNotFound(thingId string) *responseError {
	return generateError(ErrCodeThingNotFound, thingId)
}

func ErrPropertyOfThingNotFound(property, propertySet, thingId string) *responseError {
	return generateError(ErrCodePropertyOfThingNotFound, property, propertySet, thingId)
}

func ErrVirtualParameterOverlapsOperand(vp string) *responseError {
	return generateError(ErrCodeVirtualParameterOverlapsOperand, vp)
}

func ErrVirtualParameterMismatchesProperty(name string, vpdt, pdt v1.Datatype, vpu, pu string, vplen, plen int) *responseError {
	return generateError(ErrCodeVirtualParameterMismatchesProperty, name, vpdt, pdt, vpu, pu, vplen, plen)
}

func ErrTemplateNotFound(tmpId string) *responseError {
	return generateError(ErrCodeTemplateMessageIdNotFound, tmpId)
}

func ErrParentTypeNotFound(parentType string) *responseError {
	return generateError(ErrCodeParentTypeNotFound, parentType)
}

func ErrPropertySetTypeNotFound(propertySetType string) *responseError {
	return generateError(ErrCodePropertySetTypeNotFound, propertySetType)
}

func ErrThingTypeNotFound(thingType string) *responseError {
	return generateError(ErrCodeThingTypeNotFound, thingType)
}

func ErrAgentTypeNotFound(agentType string) *responseError {
	return generateError(ErrCodeAgentTypeNotFound, agentType)
}

func ErrAgentNotFound(agent string) *responseError {
	return generateError(ErrCodeAgentNotFound, agent)
}

func ErrDataSourceNotFound(dataSource string) *responseError {
	return generateError(ErrCodeDataSourceNotFound, dataSource)
}

func ErrDataPointNotFound(dataPoint string) *responseError {
	return generateError(ErrCodeDataPointNotFound, dataPoint)
}

func ErrPropertySetNotFound(propertySet string) *responseError {
	return generateError(ErrCodePropertySetNotFound, propertySet)
}

func ErrPropertyNotFound(property string) *responseError {
	return generateError(ErrCodePropertyNotFound, property)
}

func ErrCommandOptionExists(option string) *responseError {
	return generateError(ErrCodeCommandOptionExists, option)
}

func ErrCommandTypeNotFound(commandTypeId string) *responseError {
	return generateError(ErrCodeCommandTypeNotFound, commandTypeId)
}

func ErrCommandOptionNotFound(options []string) *responseError {
	return generateError(ErrCodeCommandOptionNotFound, strings.Join(options, "|"))
}

func ErrCommandOptionValueMissed(options []string) *responseError {
	return generateError(ErrCodeCommandOptionValueMissed, strings.Join(options, "|"))
}

func ErrIntegerInvalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeIntegerInvalid, convert(infos)...)
}

func ErrLongInvalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeLongInvalid, convert(infos)...)
}

func ErrDoubleInvalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeDoubleInvalid, convert(infos)...)
}

func ErrBooleanInvalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeBooleanInvalid, convert(infos)...)
}

func ErrTimestampInvalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeTimestampInvalid, convert(infos)...)
}

func ErrStringTooLong(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeStringTooLong, convert(infos)...)
}

func ErrEnumInvalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeEnumInvalid, convert(infos)...)
}

func ErrMapInvalid(infos ...string) *responseError {
	if len(infos) == 1 {
		infos = append(infos, "")
	}
	return generateError(ErrCodeMapInvalid, convert(infos)...)
}

func ErrLinkInvalid(link string) *responseError {
	return generateError(ErrCodeLinkInvalid, link)
}

func ErrPropertyNotWritable(property string) *responseError {
	return generateError(ErrCodePropertyNotWritable, property)
}

func ErrResourceDeleting(resource string) *responseError {
	return generateError(ErrCodeResourceDeleting, resource)
}

func ErrAssociatedResourceDeletionFailed(resource string) *responseError {
	return generateError(ErrCodeAssociatedResourceDeletionFailed, resource)
}

func ErrTimeInvalid(time string) *responseError {
	return generateError(ErrCodeTimeInvalid, time)
}

func ErrDuplicatedComponent(component string) *responseError {
	return generateError(ErrCodeDuplicatedComponent, component)
}

func ErrComponentUriInvalid(component string, uri string) *responseError {
	return generateError(ErrCodeComponentUriInvalid, component, uri)
}

func ErrChildTypeExist(typeId string) *responseError {
	return generateError(ErrCodeChildTypeExist, typeId)
}

func ErrAssociateResourceExist(resourceId string) *responseError {
	return generateError(ErrCodeAssociateResourceExist, resourceId)
}

func ErrRollupIntervalInvalid(parameter string) *responseError {
	return generateError(ErrCodeRollupIntervalInvalid, parameter)
}

func ErrRollupStartInvalid(parameter string) *responseError {
	return generateError(ErrCodeRollupStartInvalid, parameter)
}

func ErrRollupEndInvalid(parameter string) *responseError {
	return generateError(ErrCodeRollupEndInvalid, parameter)
}

func ErrDuplicatedField(field string) *responseError {
	return generateError(ErrCodeDuplicatedField, field)
}

func ErrFieldExists(field, parents string) *responseError {
	return generateError(ErrCodeFieldExists, field, parents)
}

func ErrRequiredFieldMissed(fields []string) *responseError {
	return generateError(ErrCodeRequiredFieldMissed, strings.Join(fields, "|"))
}

func ErrFieldInvalid(field string, value interface{}) *responseError {
	return generateError(ErrCodeFieldInvalid, value, field)
}

func ErrFieldUnSupported(t string) *responseError {
	return generateError(ErrCodeFieldUnSupported, t)
}

func ErrDeleteFundamentalResourceFailed(resource string) *responseError {
	return generateError(ErrCodeDeleteFundamentalResourceFailed, resource)
}

func ErrInvalidValue(info error) *responseError {
	s, _ := json.Marshal(info)
	return generateError(ErrCodeInvalidValue, s)
}

func ErrInvalidUpdate(info error) *responseError {
	s, _ := json.Marshal(info)
	return generateError(ErrCodeInvalidUpdate, s)
}

// func ErrImmutable(paths []string) *responseError {
// 	s, _ := json.Marshal(paths)
// 	return generateErrorWrapper(ErrCodeImmutable, apis.ErrImmutable, s)
// }

func ErrTooManyJsonPatchOperations(max int) *responseError {
	return generateError(ErrCodeTooManyJsonPatchOperations, max)
}

func ErrCharacteristicNotFound(char string) *responseError {
	return generateError(ErrCodeCharacteristicNotFound, char)
}

func ErrCycledThing(thingId string) *responseError {
	return generateError(ErrCodeCycledThing, thingId)
}
