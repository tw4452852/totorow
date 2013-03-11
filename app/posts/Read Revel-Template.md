Read Revel - Template | 2012-12-17
# Read Revel - Template

Did you ever remember the previous chapter. The template occurs many times.It almost handle all the
html pages, such as errors, regular home page and so on.

Revel uses go official template, also a similar way to maintain the templates. It abstract a
template loader. It is like a small database.

![template_loader](/public/images/read_revel/template_loader.png)

## Structure - templateLoader

`templateLoader` object handles loading and parsing of templates.

~~~ {prettyprint}
type TemplateLoader struct {
	// This is the set of all templates under views
	templateSet *template.Template
	// If an error was encountered parsing the templates, it is stored here.
	compileError *Error
	// Paths to search for templates, in priority order.
	paths []string
	// Map from template name to the path from whence it was loaded.
	templatePaths map[string]string
}
~~~

### Method - NewTemplateLoader
To build a template loader , the only thing you should prepare is the base path of searching.

~~~ {prettyprint}
func NewTemplateLoader(paths []string) *TemplateLoader {
	loader := &TemplateLoader{
		paths: paths,
	}
	return loader
}
~~~

### Method - Refresh
This method scans the views directory and parses all templates as Go templates

It go through every file under base path and builds up a template set in the end.

~~~ {prettyprint}
func (loader *TemplateLoader) Refresh() *Error {
	TRACE.Println("Refresh")
	loader.compileError = nil
	loader.templatePaths = map[string]string{}

	// Walk through the template loader's paths and build up a template set.
	var templateSet *template.Template = nil
	for _, basePath := range loader.paths {

		// Walk only returns an error if the template loader is completely unusable
		// (namely, if one of the TemplateFuncs does not have an acceptable signature).
		funcErr := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			...
		})

		// If there was an error with the Funcs, set it and return immediately.
		if funcErr != nil {
			loader.compileError = funcErr.(*Error)
			return loader.compileError
		}
	}

	// Note: compileError may or may not be set.
	loader.templateSet = templateSet
	return loader.compileError
}
~~~

The callback function is to read every template file content and add it into template set

~~~ {prettyprint lang-c}
			if err != nil {
				ERROR.Println("error walking templates:", err)
				return nil
			}

			// Walk into directories.
			if info.IsDir() {
				if !loader.WatchDir(info) {
					return filepath.SkipDir
				}
				return nil
			}

			if !loader.WatchFile(info.Name()) {
				return nil
			}

			// Convert template names to use forward slashes, even on Windows.
			templateName := path[len(basePath)+1:]
			if os.PathSeparator == '\\' {
				templateName = strings.Replace(templateName, `\`, `/`, -1)
			}

			// If we already loaded a template of this name, skip it.
			if _, ok := loader.templatePaths[templateName]; ok {
				return nil
			}
			loader.templatePaths[templateName] = path

			fileBytes, err := ioutil.ReadFile(path)
			if err != nil {
				ERROR.Println("Failed reading file:", path)
				return nil
			}

			fileStr := string(fileBytes)
			if templateSet == nil {
				// Create the template set.  This panics if any of the funcs do not
				// conform to expectations, so we wrap it in a func and handle those
				// panics by serving an error page.
				var funcError *Error
				func() {
					defer func() {
						if err := recover(); err != nil {
							funcError = &Error{
								Title:       "Panic (Template Loader)",
								Description: fmt.Sprintln(err),
							}
						}
					}()
					templateSet = template.New(templateName).Funcs(TemplateFuncs)
					_, err = templateSet.Parse(fileStr)
				}()

				if funcError != nil {
					return funcError
				}

			} else {
				_, err = templateSet.New(templateName).Parse(fileStr)
			}

			// Store / report the first error encountered.
			if err != nil && loader.compileError == nil {
				line, description := parseTemplateError(err)
				loader.compileError = &Error{
					Title:       "Template Compilation Error",
					Path:        templateName,
					Description: description,
					Line:        line,
					SourceLines: strings.Split(fileStr, "\n"),
				}
				ERROR.Printf("Template compilation error (In %s around line %d):\n%s",
					templateName, line, description)
			}
			return nil
~~~
- files and dirs beginning with "." will be ignored.
- if there is template parsing err happened, it will be logged in `compileError`. Even there is a
  error, the template set is still generated.

### Method - Template
Return the template with given name. The name is the template's path relative to a template loader
root

~~~ {prettyprint}
func (loader *TemplateLoader) Template(name string) (Template, error) {
	// Look up and return the template.
	tmpl := loader.templateSet.Lookup(name)

	// This is necessary.
	// If a nil loader.compileError is returned directly, a caller testing against
	// nil will get the wrong result.  Something to do with casting *Error to error.
	var err error
	if loader.compileError != nil {
		err = loader.compileError
	}

	if tmpl == nil && err == nil {
		return nil, fmt.Errorf("Template %s not found.", name)
	}

	return GoTemplate{tmpl, loader}, err
}
~~~

## Structure - GoTemplate

To overwrite go standard template, it wrapped go `*template.TemplateLoader`.
~~~ {prettyprint}
type GoTemplate struct {
	*template.Template
	loader *TemplateLoader
}
~~~

And it has two methods for satisfying `revel.Template` interface.

~~~ {prettyprint}
type Template interface {
	Name() string
	Content() []string
	Render(wr io.Writer, arg interface{}) error
}
~~~
~~~ {prettyprint}
func (gotmpl GoTemplate) Render(wr io.Writer, arg interface{}) error {
	return gotmpl.Execute(wr, arg)
}

func (gotmpl GoTemplate) Content() []string {
	content, _ := ReadLines(gotmpl.loader.templatePaths[gotmpl.Name()])
	return content
}
~~~

## Template functions

Revel predefined some itself template functions to the go standard template functions, it help the
user to write a sample template file.

### function - eq
a simple `"a == b"` test.

Usage:

~~~ {prettyprint}
<div class="message {{if eq .User "you"}}you{{end}}">
~~~
The `eq` function:

~~~ {prettyprint}
"eq":  func(a, b interface{}) bool { return a == b },
~~~

### function - set

Set a variable in the given context.

Usage:

~~~ {prettyprint}
{{set . "title" "Basic Chat room"}}

<h1>{{.title}}</h1>
~~~
The `set` function:

~~~ {prettyprint}
"set": func(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
	renderArgs[key] = value
	return template.HTML("")
},
~~~

### function - append

Add a variable to an array, or start an array, in the given context.

Usage:

~~~ {prettyprint}
{{append . "moreScripts" "js/jquery-ui.js"}}

{{range .moreScripts}}
	<link rel="stylesheet" type="text/css" href="/public/{{.}}">
{{end}}
~~~

The `append` function:

~~~ {prettyprint}
"append": func(renderArgs map[string]interface{}, key string, value interface{}) template.HTML {
	if renderArgs[key] == nil {
		renderArgs[key] = []interface{}{value}
	} else {
		renderArgs[key] = append(renderArgs[key].([]interface{}), value)
	}
	return template.HTML("")
},
~~~

### function - field

A helper for input fields

Usage:

~~~ {prettyprint}
{{with $field := field "booking.CheckInDate" .}}
	<p class = "error">
	<strong>Check In Date:</strong>
	<input type="text" size="10" name="{{$field.Name}}" class="datepicker" vaule="">
	* <span class="error">{{$field.Error}}</span>
	</p>
{{end}}
~~~

The `field` function:

~~~ {prettyprint linenums}
"field": func(name string, renderArgs map[string]interface{}) *Field {
	value, _ := renderArgs["flash"].(map[string]string)[name]
	err, _ := renderArgs["errors"].(map[string]*ValidationError)[name]
	return &Field{
		Name:  name,
		Value: value,
		Error: err,
	}
},
~~~

The `field` structure:

~~~ {prettyprint}
type Field struct {
	Name, Value string
	Error       *ValidationError
}
~~~

It has two helper methods.

~~~ {prettyprint}
func (f *Field) ErrorClass() string {
	if f.Error != nil {
		return ERROR_CLASS
	}
	return ""
}

// Return "checked" if this field.Value matches the provided value
func (f *Field) Checked(val string) string {
	if f.Value == val {
		return "checked"
	}
	return ""
}
~~~

### function - option

Assists in constructing HTML `option` elements, in conjunction with the field helper.

Usage:

~~~ {prettyprint}
{{with $field := field "booking.Beds" .}}
<select name="{{$field.Name}}">
	{{option $field "1" "One king-size bed"}}
	{{option $field "2" "Two double beds"}}
	{{option $field "3" "Three beds"}}
</select>
{{end}}
~~~

The `option` function:

~~~ {prettyprint}
"option": func(f *Field, val, label string) template.HTML {
	selected := ""
	if f.Value == val {
		selected = " selected"
	}
	return template.HTML(fmt.Sprintf(`<option value="%s"%s>%s</option>`, html.EscapeString(val), selected, html.EscapeString(label)))
},
~~~

### function - radio

Assists in constructing HTML radio input elements, in conjunction with the field helper.

Usage:

~~~ {prettyprint}
{{with $field := field "booking.Smoking" .}}
	{{radio $field "true"}} Smoking
	{{radio $field "false"}} Non smoking
{{end}}
~~~

The `radio` function:

~~~ {prettyprint linenums}
"radio": func(f *Field, val string) template.HTML {
	checked := ""
	if f.Value == val {
		checked = " checked"
	}
	return template.HTML(fmt.Sprintf(`<input type="radio" name="%s" value="%s"%s>`,
		html.EscapeString(f.Name), html.EscapeString(val), checked))
},
~~~

### function - pad

Pads the given string with &nbsp;'s up to given width

Usage:

~~~ {prettyprint}
<h1>{{pad "hello" 10}}</h1>
~~~

The `pad` function:

~~~ {prettyprint}
"pad": func(str string, width int) template.HTML {
	if len(str) >= width {
		return template.HTML(html.EscapeString(str))
	}
	return template.HTML(html.EscapeString(str) + strings.Repeat("&nbsp;", width-len(str)))
},
~~~

### function - errorClass

If there was an error, it output the literal string "error", else "".

Usage:

~~~ {prettyprint}
<p class={{errorClass "error" .}}>
<h2>Some error happend</h2>
</p>
~~~

The `errorClass` function:

~~~ {prettyprint linenums}
"errorClass": func(name string, renderArgs map[string]interface{}) template.HTML {
	errorMap, ok := renderArgs["errors"].(map[string]*ValidationError)
	if !ok {
		WARN.Println("Called 'errorClass' without 'errors' in the render args.")
		return template.HTML("")
	}
	valError, ok := errorMap[name]
	if !ok || valError == nil {
		return template.HTML("")
	}
	return template.HTML(ERROR_CLASS)
},
~~~

### function - url

Return a url capable of invoking a given controller method:
`"Application.ShowApp 123" => "/app/123"`

Usage:

~~~ {prettyprint}
<a href={{url "Application.ShowApp" "123"}}> Apps </a>
~~~

The `url` function:

~~~ {prettyprint}
"url": ReverseUrl,
~~~
~~~ {prettyprint}
func ReverseUrl(args ...interface{}) string {
	if len(args) == 0 {
		ERROR.Println("Warning: no arguments provided to url function")
		return "#"
	}

	action := args[0].(string)
	actionSplit := strings.Split(action, ".")
	var ctrl, meth string
	if len(actionSplit) != 2 {
		ERROR.Println("Warning: Must provide Controller.Method for reverse router.")
		return "#"
	}
	ctrl, meth = actionSplit[0], actionSplit[1]
	controllerType := LookupControllerType(ctrl)
	methodType := controllerType.Method(meth)
	argsByName := make(map[string]string)
	for i, argValue := range args[1:] {
		argsByName[methodType.Args[i].Name] = fmt.Sprintf("%s", argValue)
	}

	return MainRouter.Reverse(args[0].(string), argsByName).Url
}
~~~
Extract `controller` and `method` from the first string. Assert a lookup in router adding the exist args.

FIN.
