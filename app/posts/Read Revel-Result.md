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

~~~ {prettyprint linenums:97}
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

~~~ {prettyprint linenums:27}
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
