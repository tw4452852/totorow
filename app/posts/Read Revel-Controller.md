Read Revel - Controller | 2012-12-10
# Read Revel - Controller

The Revel makes it easy to build web applications using the Model-View-Controller(MVC) pattern by
relying on conventions that require a certain structure in your application.

There is a explanation about MVC in Revel official site:

> - Models are the essential data objects that describe your application domain. Models also contain
> - domain-specific logic for querying and updating the data. Views describe how data is presented and manipulated. In our case, this is the template that is used to present data and controls to the user.
> - Controllers handle the request execution. They perform the user’s desired action, they decide which View to display, and they prepare and provide the necessary data to the View for rendering.

## Concepts

In order to explain the controller, we must introduce some concepts firstly.

### Flash
Flash represents a cookie that gets overwritten on each request.
It allows data to be stored across one page at a time.
This is commonly used to implement success or error messages.

~~~ {prettyprint}
type Flash struct {
	Data, Out map[string]string
}
~~~

Before a request, it is generated with the cookies in the request.

~~~ {prettyprint}
func (p FlashPlugin) BeforeRequest(c *Controller) {
	c.Flash = restoreFlash(c.Request.Request)
	c.RenderArgs["flash"] = c.Flash.Data
}
~~~
~~~ {prettyprint}
func restoreFlash(req *http.Request) Flash {
	flash := Flash{
		Data: make(map[string]string),
		Out:  make(map[string]string),
	}
	if cookie, err := req.Cookie(CookiePrefix + "_FLASH"); err == nil {
		ParseKeyValueCookie(cookie.Value, func(key, val string) {
			flash.Data[key] = val
		})
	}
	return flash
}
~~~

And after the request, set the cookies according to the output map.

~~~ {prettyprint}
func (p FlashPlugin) AfterRequest(c *Controller) {
	// Store the flash.
	var flashValue string
	for key, value := range c.Flash.Out {
		flashValue += "\x00" + key + ":" + value + "\x00"
	}
	c.SetCookie(&http.Cookie{
		Name:  CookiePrefix + "_FLASH",
		Value: url.QueryEscape(flashValue),
		Path:  "/",
	})
}
~~~

By the way, the flash predefine two methods that export msg to output map.
They are `Error` and `Success`.

~~~ {prettyprint}
func (f Flash) Error(msg string, args ...interface{}) {
	if len(args) == 0 {
		f.Out["error"] = msg
	} else {
		f.Out["error"] = fmt.Sprintf(msg, args...)
	}
}

func (f Flash) Success(msg string, args ...interface{}) {
	if len(args) == 0 {
		f.Out["success"] = msg
	} else {
		f.Out["success"] = fmt.Sprintf(msg, args...)
	}
}
~~~

### Session
Session is same as the flash, omit it.

### Request

A http request is abstracted to a `Request` structure:

~~~ {prettyprint}
type Request struct {
	*http.Request
	ContentType string
	Format      string // "html", "xml", "json", or "text"
}
~~~
A embedding struct of `*http.Request` adding `ContentType` and `Format` attribute.

There is method to new a Request:

~~~ {prettyprint}
func NewRequest(r *http.Request) *Request {
	return &Request{
		Request:     r,
		ContentType: ResolveContentType(r),
		Format:      ResolveFormat(r),
	}
}
~~~

`ResolveContentType` is to extract the content type from the http header.

~~~ {prettyprint}
// Get the content type.
// e.g. From "multipart/form-data; boundary=--" to "multipart/form-data"
// If none is specified, returns "text/html" by default.
func ResolveContentType(req *http.Request) string {
	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		return "text/html"
	}
	return strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
}
~~~

`ResolveFormat` is same as the `ResolveContentType`.

~~~ {prettyprint}
func ResolveFormat(req *http.Request) string {
	accept := req.Header.Get("accept")

	switch {
	case accept == "",
		strings.HasPrefix(accept, "*/*"),
		strings.Contains(accept, "application/xhtml"),
		strings.Contains(accept, "text/html"):
		return "html"
	case strings.Contains(accept, "application/xml"),
		strings.Contains(accept, "text/xml"):
		return "xml"
	case strings.Contains(accept, "text/plain"):
		return "txt"
	case strings.Contains(accept, "application/json"),
		strings.Contains(accept, "text/javascript"):
		return "json"
	}

	return "html"
}
~~~

### Response

There is also a Response struct to abstract the http response.

~~~ {prettyprint}
type Response struct {
	Status      int
	ContentType string

	Out http.ResponseWriter
}
~~~
And a new method to create it.

~~~ {prettyprint}
func NewResponse(w http.ResponseWriter) *Response {
	return &Response{Out: w}
}
~~~

A method to set `Status` and `ContentType` fields:

~~~ {prettyprint}
// Write the header (for now, just the status code).
// The status may be set directly by the application (c.Response.Status = 501).
// if it isn't, then fall back to the provided status code.
func (resp *Response) WriteHeader(defaultStatusCode int, defaultContentType string) {
	if resp.Status == 0 {
		resp.Status = defaultStatusCode
	}
	if resp.ContentType == "" {
		resp.ContentType = defaultContentType
	}
	resp.Out.Header().Set("Content-Type", resp.ContentType)
	resp.Out.WriteHeader(resp.Status)
}
~~~

## Management

To manage all kinds of controllers, Revel use `ControllerType` to express a controller instance in
his internal bookkeeping.

~~~ {prettyprint}
type ControllerType struct {
	Type    reflect.Type
	Methods []*MethodType
}

type MethodType struct {
	Name           string
	Args           []*MethodArg
	RenderArgNames map[int][]string
	lowerName      string
}

type MethodArg struct {
	Name string
	Type reflect.Type
}
~~~

To see whether a method is belong a controller, `ControllerType` export a `Method` method.

~~~ {prettyprint}
// Searches for a given exported method (case insensitive)
func (ct *ControllerType) Method(name string) *MethodType {
	lowerName := strings.ToLower(name)
	for _, method := range ct.Methods {
		if method.lowerName == lowerName {
			return method
		}
	}
	return nil
}
~~~
just compare their names.

And all `ControllerType` are collected in a map variable `controllers`. If you want to add a new
controller, use the `RegisterController` function to do it.

~~~ {prettyprint}
// Register a Controller and its Methods with Revel.
func RegisterController(c interface{}, methods []*MethodType) {
	// De-star the controller type
	// (e.g. given TypeOf((*Application)(nil)), want TypeOf(Application))
	var t reflect.Type = reflect.TypeOf(c)
	var elem reflect.Type = t.Elem()

	// De-star all of the method arg types too.
	for _, m := range methods {
		m.lowerName = strings.ToLower(m.Name)
		for _, arg := range m.Args {
			arg.Type = arg.Type.Elem()
		}
	}

	controllers[strings.ToLower(elem.Name())] = &ControllerType{Type: elem, Methods: methods}
	TRACE.Printf("Registered controller: %s", elem.Name())
}
~~~
- Note: the controller itself and methods are all pointers, so we must dereference them at first.

There is also a helper function `LookupControllerType` to find a controller in the map.

~~~ {prettyprint}
func LookupControllerType(name string) *ControllerType {
	return controllers[strings.ToLower(name)]
}
~~~

When user need create their own controllers, it may inherit revel base controller. So revel used two
abstractions to express them respectively.

A user controller may like this:

~~~ {prettyprint}
type AppController struct {
	*rev.Controller
	...
	//some other user fields
}
~~~
And `rev.Controller` we will talk it later. `NewAppController` function will create a
`AppController`, Note that you must firstly register the controller, then you can create it.

~~~ {prettyprint}
func NewAppController(req *Request, resp *Response, controllerName, methodName string) (*Controller, reflect.Value) {
	var appControllerType *ControllerType = LookupControllerType(controllerName)
	if appControllerType == nil {
		INFO.Printf("Controller %s not found: %s", controllerName, req.URL)
		return nil, reflect.ValueOf(nil)
	}

	controller := NewController(req, resp, appControllerType)
	appControllerPtr := initNewAppController(appControllerType.Type, controller)
~~~
And `initNewAppController` is a helper that initializes (zeros) a new app controller value.
Generally, everything is set to its zero value, except:

- Embedded controller pointers are newed up.
- The rev.Controller embedded type is set to the value provided.

~~~ {prettyprint}
func initNewAppController(appControllerType reflect.Type, c *Controller) reflect.Value {
	// It might be a multi-level embedding, so we have to create new controllers
	// at every level of the hierarchy.
	// ASSUME: the first field in each type is the way up to rev.Controller.
	appControllerPtr := reflect.New(appControllerType)
	ptr := appControllerPtr
	for {
		var (
			embeddedField     reflect.Value = ptr.Elem().Field(0)
			embeddedFieldType reflect.Type  = embeddedField.Type()
		)

		// Check if it's the controller.
		if embeddedFieldType == controllerType {
			embeddedField.Set(reflect.ValueOf(c).Elem())
			break
		} else if embeddedFieldType == controllerPtrType {
			embeddedField.Set(reflect.ValueOf(c))
			break
		}

		// If the embedded field is a pointer, then instantiate an object and set it.
		// (If it's not a pointer, then it's already initialized)
		if embeddedFieldType.Kind() == reflect.Ptr {
			embeddedField.Set(reflect.New(embeddedFieldType.Elem()))
			ptr = embeddedField
		} else {
			ptr = embeddedField.Addr()
		}
	}
	return appControllerPtr
}
~~~
Here, there is assume that the `rev.Controller` or `*rev.Controller` is always lays at the first
field of the container, no matter how many levels it belongs. Fields initialed a zero value if it is
not the destination.

## Structure - Result

A Result express a http response usually.

~~~ {prettyprint}
type Result interface {
	Apply(req *Request, resp *Response)
}
~~~

## Structure - Controller

Controller is the context for the request. It contains the request and responese data. So it the
central structure of the netflow.

~~~ {prettyprint}
type Controller struct {
	Name       string
	Type       *ControllerType
	MethodType *MethodType

	Request  *Request
	Response *Response

	Flash      Flash                  // User cookie, cleared after each request.
	Session    Session                // Session, stored in cookie, signed.
	Params     *Params                // Parameters from URL and form (including multipart).
	Args       map[string]interface{} // Per-request scratch space.
	RenderArgs map[string]interface{} // Args passed to the template.
	Validation *Validation            // Data validation helpers
	Txn        *sql.Tx                // Nil by default, but may be used by the app / plugins
}
~~~
Some fields we don't talk today, we are concern the structure itself now.

### Method - NewController

~~~ {prettyprint}
func NewController(req *Request, resp *Response, ct *ControllerType) *Controller {
	return &Controller{
		Name:     ct.Type.Name(),
		Type:     ct,
		Request:  req,
		Response: resp,
		Params:   ParseParams(req),
		Args:     map[string]interface{}{},
		RenderArgs: map[string]interface{}{
			"RunMode": RunMode,
		},
	}
}
~~~
`ParseParams` is to extract the parameters from the request.

### Method - Invoke
Invoke the given method, save headers/cookies to the response, and apply the. Definition is here.

~~~ {prettyprint}
func (c *Controller) Invoke(appControllerPtr reflect.Value, method reflect.Value, methodArgs []reflect.Value)
~~~

Firstly, register two defer functions. One for handle panic and one for clean up some temporary stuffs.

~~~ {prettyprint}
// Handle panics.
defer func() {
	if err := recover(); err != nil {
		handleInvocationPanic(c, err)
	}
}()

// Clean up from the request.
defer func() {
	// Delete temp files.
	if c.Request.MultipartForm != nil {
		err := c.Request.MultipartForm.RemoveAll()
		if err != nil {
			WARN.Println("Error removing temporary files:", err)
		}
	}

	for _, tmpFile := range c.Params.tmpFiles {
		err := os.Remove(tmpFile.Name())
		if err != nil {
			WARN.Println("Could not remove upload temp file:", err)
		}
	}
}()
~~~

Then the sequence is:

~~~ {prettyprint}
// Run the plugins.
plugins.BeforeRequest(c)

// Calculate the Result by running the interceptors and the action.
resultValue := func() reflect.Value {
	// Call the BEFORE interceptors
	result := c.invokeInterceptors(BEFORE, appControllerPtr)
	if result != nil {
		return reflect.ValueOf(result)
	}

	// Invoke the action.
	resultValue := method.Call(methodArgs)[0]

	// Call the AFTER interceptors
	result = c.invokeInterceptors(AFTER, appControllerPtr)
	if result != nil {
		return reflect.ValueOf(result)
	}
	return resultValue
}()

plugins.AfterRequest(c)

if resultValue.Kind() == reflect.Interface && resultValue.IsNil() {
	return
}
result := resultValue.Interface().(Result)

// Apply the result, which generally results in the ResponseWriter getting written.
result.Apply(c.Request, c.Response)
~~~
So the work flow is like this:

![invoke_flow](/public/images/read_revel/invoke_flow.png)

### Method - Render
Render a template corresponding to the calling Controller method.

At first, it get method itself and save it in `controller.MethodType`.

~~~ {prettyprint}
func (c *Controller) Render(extraRenderArgs ...interface{}) Result {
	// Get the calling function name.
	pc, _, line, ok := runtime.Caller(1)
	if !ok {
		ERROR.Println("Failed to get Caller information")
		return nil
	}
	// e.g. sample/app/controllers.(*Application).Index
	var fqViewName string = runtime.FuncForPC(pc).Name()
	var viewName string = fqViewName[strings.LastIndex(fqViewName, ".")+1 : len(fqViewName)]

	// Determine what method we are in.
	// (e.g. the invoked controller method might have delegated to another method)
	methodType := c.MethodType
	if methodType.Name != viewName {
		methodType = c.Type.Method(viewName)
		if methodType == nil {
			return c.RenderError(fmt.Errorf(
				"No Method %s in Controller %s when loading the view."+
					" (delegating Render is only supported within the same controller)",
				viewName, c.Name))
		}
	}
~~~

Then we should set the render variables. Because the Render method may be called from different
places, we use line number to distinguish each other.

~~~ {prettyprint}
// Get the extra RenderArgs passed in.
if renderArgNames, ok := methodType.RenderArgNames[line]; ok {
	if len(renderArgNames) == len(extraRenderArgs) {
		for i, extraRenderArg := range extraRenderArgs {
			c.RenderArgs[renderArgNames[i]] = extraRenderArg
		}
	} else {
		ERROR.Println(len(renderArgNames), "RenderArg names found for",
			len(extraRenderArgs), "extra RenderArgs")
	}
} else {
	ERROR.Println("No RenderArg names found for Render call on line", line,
		"(Method", methodType, ", ViewName", viewName, ")")
}

return c.RenderTemplate(c.Name + "/" + viewName + ".html")
~~~

### Method - RenderTemplate
A less magical way to render a template. Renders the given template, using the current `controller.RenderArgs`.

~~~ {prettyprint}
func (c *Controller) RenderTemplate(templatePath string) Result {

	// Get the Template.
	template, err := MainTemplateLoader.Template(templatePath)
	if err != nil {
		return c.RenderError(err)
	}

	return &RenderTemplateResult{
		Template:   template,
		RenderArgs: c.RenderArgs,
	}
}
~~~
The request template is loaded by a `templateLoader`.

### Method - RenderJson
Same as `RenderTemplate`

~~~ {prettyprint}
// Uses encoding/json.Marshal to return JSON to the client.
func (c *Controller) RenderJson(o interface{}) Result {
	return RenderJsonResult{o}
}
~~~

### Method - RenderXml
Same as `RenderTemplate`

~~~ {prettyprint}
// Uses encoding/xml.Marshal to return XML to the client.
func (c *Controller) RenderXml(o interface{}) Result {
	return RenderXmlResult{o}
}
~~~

### Method - RenderText
Same as `RenderTemplate`

~~~ {prettyprint}
// Render plaintext in response, printf style.
func (c *Controller) RenderText(text string, objs ...interface{}) Result {
	finalText := text
	if len(objs) > 0 {
		finalText = fmt.Sprintf(text, objs)
	}
	return &RenderTextResult{finalText}
}
~~~

### Method - Todo
Same as `RenderTemplate`

~~~ {prettyprint}
// Render a "todo" indicating that the action isn't done yet.
func (c *Controller) Todo() Result {
	c.Response.Status = http.StatusNotImplemented
	return c.RenderError(&Error{
		Title:       "TODO",
		Description: "This action is not implemented",
	})
}
~~~

### Method - NotFound
Same as `RenderTemplate`

~~~ {prettyprint}
func (c *Controller) NotFound(msg string) Result {
	c.Response.Status = http.StatusNotFound
	return c.RenderError(&Error{
		Title:       "Not Found",
		Description: msg,
	})
}
~~~

### Method - RenderFile
Return a file, either displayed inline or downloaded as an attachment.
The name and size are taken from the file info.

~~~ {prettyprint}
func (c *Controller) RenderFile(file *os.File, delivery ContentDisposition) Result {
	var length int64 = -1
	fileInfo, err := file.Stat()
	if err != nil {
		WARN.Println("RenderFile error:", err)
	}
	if fileInfo != nil {
		length = fileInfo.Size()
	}
	return &BinaryResult{
		Reader:   file,
		Name:     filepath.Base(file.Name()),
		Length:   length,
		Delivery: delivery,
	}
}
~~~

### Method - Redirect
Redirect to an action or to a URL.

~~~ {prettyprint}
func (c *Controller) Redirect(val interface{}, args ...interface{}) Result {
	if url, ok := val.(string); ok {
		if len(args) == 0 {
			return &RedirectToUrlResult{url}
		}
		return &RedirectToUrlResult{fmt.Sprintf(url, args...)}
	}
	return &RedirectToActionResult{val}
}
~~~
