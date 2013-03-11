Read Revel - Result | 2012-12-10
# Read Revel - Result
Following the previous chapter, we continue to talk about kinds of Result that revel has
accomplished.

Let's start with a simple one

## Structure - PlaintextErrorResult

Definition is:

~~~ {prettyprint}
type PlaintextErrorResult struct {
	Error error
}
~~~

Its Apply method is:

~~~ {prettyprint}
// This method is used when the template loader or error template is not available.
func (r PlaintextErrorResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusInternalServerError, "text/plain")
	resp.Out.Write([]byte(r.Error.Error()))
}
~~~
Just set status code and error string.

## structures - ErrorResult

This result handles all kinds of error codes (500, 404, ..).
It renders the relevant error page (errors/CODE.format, e.g. errors/500.json).

~~~ {prettyprint}
type ErrorResult struct {
	RenderArgs map[string]interface{}
	Error      error
}
~~~

And its `Apply` method. Firstly get error template.

~~~ {prettyprint}
format := req.Format
status := resp.Status
if status == 0 {
	status = http.StatusInternalServerError
}

contentType := ContentTypeByFilename("xxx." + format)
if contentType == DefaultFileContentType {
	contentType = "text/plain"
}

// Get the error template.
var err error
templatePath := fmt.Sprintf("errors/%d.%s", status, format)
tmpl, err := MainTemplateLoader.Template(templatePath)
~~~

If template is not found, use `PlaintextErrorResult` to show the error info.

~~~ {prettyprint}
showPlaintext := func(err error) {
	PlaintextErrorResult{fmt.Errorf("Server Error:\n%s\n\n"+
		"Additionally, an error occurred when rendering the error page:\n%s",
		r.Error, err)}.Apply(req, resp)
}

if tmpl == nil {
	if err == nil {
		err = fmt.Errorf("Couldn't find template %s", templatePath)
	}
	showPlaintext(err)
	return
}
~~~

At last, render the template and push the result into the http response.

~~~ {prettyprint}
// If it's not a revel error, wrap it in one.
var revelError *Error
switch e := r.Error.(type) {
case *Error:
	revelError = e
case error:
	revelError = &Error{
		Title:       "Server Error",
		Description: e.Error(),
	}
}

if revelError == nil {
	panic("no error provided")
}

r.RenderArgs["RunMode"] = RunMode
r.RenderArgs["Error"] = revelError
r.RenderArgs["Router"] = MainRouter

// Render it.
var b bytes.Buffer
err = tmpl.Render(&b, r.RenderArgs)

// If there was an error, print it in plain text.
if err != nil {
	showPlaintext(err)
	return
}

resp.WriteHeader(status, contentType)
b.WriteTo(resp.Out)
~~~

## Structure - RenderHtmlResult

This just handle the html string directly.

~~~ {prettyprint}
type RenderHtmlResult struct {
	html string
}
~~~
~~~ {prettyprint}
func (r RenderHtmlResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/html")
	resp.Out.Write([]byte(r.html))
}
~~~

## Structure - RenderJsonResult

This handle `application/json`. Just use `json.MarshalIndent` or `json.Marshal` according to the
configure `results.pretty`.

~~~ {prettyprint}
type RenderJsonResult struct {
	obj interface{}
}
~~~
~~~ {prettyprint}
func (r RenderJsonResult) Apply(req *Request, resp *Response) {
	var b []byte
	var err error
	if Config.BoolDefault("results.pretty", false) {
		b, err = json.MarshalIndent(r.obj, "", "  ")
	} else {
		b, err = json.Marshal(r.obj)
	}

	if err != nil {
		ErrorResult{Error: err}.Apply(req, resp)
		return
	}

	resp.WriteHeader(http.StatusOK, "application/json")
	resp.Out.Write(b)
}
~~~

## Structure - RenderXmlResult

This handle `application/xml`.

~~~ {prettyprint}
type RenderXmlResult struct {
	obj interface{}
}
~~~
~~~ {prettyprint}
func (r RenderXmlResult) Apply(req *Request, resp *Response) {
	var b []byte
	var err error
	if Config.BoolDefault("results.pretty", false) {
		b, err = xml.MarshalIndent(r.obj, "", "  ")
	} else {
		b, err = xml.Marshal(r.obj)
	}

	if err != nil {
		ErrorResult{Error: err}.Apply(req, resp)
		return
	}

	resp.WriteHeader(http.StatusOK, "application/xml")
	resp.Out.Write(b)
}
~~~

## Structure - RenderTextResult

This handle `application/plain`.

~~~ {prettyprint}
type RenderTextResult struct {
	text string
}
~~~
~~~ {prettyprint}
func (r RenderTextResult) Apply(req *Request, resp *Response) {
	resp.WriteHeader(http.StatusOK, "text/plain")
	resp.Out.Write([]byte(r.text))
}
~~~

## Structure - BinaryResult
This handle binary files. It contain the file-name ,file-length and disposition if any. All these
informations will be set in the html header.

~~~ {prettyprint}
type ContentDisposition string

var (
	Attachment ContentDisposition = "attachment"
	Inline     ContentDisposition = "inline"
)

type BinaryResult struct {
	Reader   io.Reader
	Name     string
	Length   int64
	Delivery ContentDisposition
}
~~~
~~~ {prettyprint}
func (r *BinaryResult) Apply(req *Request, resp *Response) {
	disposition := string(r.Delivery)
	if r.Name != "" {
		disposition += fmt.Sprintf("; filename=%s;", r.Name)
	}
	resp.Out.Header().Set("Content-Disposition", disposition)

	if r.Length != -1 {
		resp.Out.Header().Set("Content-Length", fmt.Sprintf("%d", r.Length))
	}
	resp.WriteHeader(http.StatusOK, ContentTypeByFilename(r.Name))
	io.Copy(resp.Out, r.Reader)
}
~~~

## Structure - RedirectToUrlResult
This handle http redirection.

~~~ {prettyprint}
type RedirectToUrlResult struct {
	url string
}
~~~
~~~ {prettyprint}
func (r *RedirectToUrlResult) Apply(req *Request, resp *Response) {
	resp.Out.Header().Set("Location", r.url)
	resp.WriteHeader(http.StatusFound, "")
}
~~~

## Structure - RedirectToActionResult
Revel supports not only redirection based on the url itself, but also based on method.

~~~ {prettyprint}
type RedirectToActionResult struct {
	val interface{}
}
~~~
~~~ {prettyprint}
func (r *RedirectToActionResult) Apply(req *Request, resp *Response) {
	url, err := getRedirectUrl(r.val)
	if err != nil {
		ERROR.Println("Couldn't resolve redirect:", err.Error())
		ErrorResult{Error: err}.Apply(req, resp)
		return
	}
	resp.Out.Header().Set("Location", url)
	resp.WriteHeader(http.StatusFound, "")
}
func getRedirectUrl(item interface{}) (string, error) {
	// Handle strings
	if url, ok := item.(string); ok {
		return url, nil
	}

	// Handle funcs
	val := reflect.ValueOf(item)
	typ := reflect.TypeOf(item)
	if typ.Kind() == reflect.Func && typ.NumIn() > 0 {
		// Get the Controller Method
		recvType := typ.In(0)
		method := FindMethod(recvType, &val)
		if method == nil {
			return "", errors.New("couldn't find method")
		}

		// Construct the action string (e.g. "Controller.Method")
		if recvType.Kind() == reflect.Ptr {
			recvType = recvType.Elem()
		}
		action := recvType.Name() + "." + method.Name
		actionDef := MainRouter.Reverse(action, make(map[string]string))
		if actionDef == nil {
			return "", errors.New("no route for action " + action)
		}

		return actionDef.String(), nil
	}

	// Out of guesses
	return "", errors.New("didn't recognize type: " + typ.String())
}
~~~
If it is a string, just same as the `RedirectToUrlResult`. Otherwise, it get the method and generate
the `controller.method` string from the router.

FIN.
