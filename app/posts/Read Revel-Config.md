Read Revel - Config | 2012-12-18
# Read Revel - Config

The application config file is named app.conf and uses the syntax accepted by
[goconfig](https://github.com/robfig/goconfig), which is similar to Microsoft INI files.

Here's an example file:

~~~ {prettyprint}
app.name=chat
app.secret=pJLzyoiDe17L36mytqC912j81PfTiolHm1veQK6Grn1En3YFdB5lvEHVTwFEaWvj
http.addr=
http.port=9000

[dev]
results.pretty=true
watch=true

log.trace.output = off
log.info.output  = stderr
log.warn.output  = stderr
log.error.output = stderr

[prod]
results.pretty=false
watch=false

log.trace.output = off
log.info.output  = off
log.warn.output  = %(app.name)s.log
log.error.output = %(app.name)s.log
~~~
Each section is a `Run Mode`. The keys at the top level(not within any section) apply to all run
modes. The key under `[prod]` section applies only to prod mode. This allows default values to be
supplied that apply across all modes, and overridden as required.

Revel uses the following properties internally:

- app.name
- app.secret - the secret key used to sign session cookies (and anywhere the application uses rev.Sign)
- http.port - the port to listen on
- http.addr - the ip address to which to bind (empty string is wildcard)
- results.pretty - RenderXml and RenderJson product nicely formatted XML/JSON.
- watch - enable source watching. if false, no watching is done regardless of other watch settings. (default True)
- watch.templates - should Revel watch for changes to views and reload? (default True)
- watch.routes - should Revel watch for changes to routes and reload? (default True)
- watch.code - should Revel watch for changes to code and reload? (default True)
- cookie.prefix - how should the Revel-produced cookies be named? (default “REVEL”)
- log.* - Logging configuration

## goconfig
As above saying, Revel configuration mode is based on `goconfig`. Let's take a look at the
`goconfig` firstly.

The configuration file consists of sections, led by a `"*[section]*"` header and followed by `"*name:
value*"` entries; `"*name=value*"` is also accepted. Note that leading whitespace is removed from
values. The optional values can contain format strings which refer to other values in the same
section, or values in a special DEFAULT section. Additional defaults can be provided on
initialization and retrieval. Comments are indicated by ";" or "#"; a comment may begin anywhere on
a line, including on the same line after parameters or section declarations.

For example:

~~~ {prettyprint}
[My Section]
foodir: %(dir)s/whatever
dir=foo
~~~
would resolve the `"*%(dir)s*"` to the value of `"*dir*"` (`*foo*` in this case). All reference expansions
are done on demand.

### Data structure
`Config` is the representation of configuration settings.

~~~ {prettyprint}
type Config struct {
	comment   string
	separator string

	// === Sections order
	lastIdSection int            // Last section identifier
	idSection     map[string]int // Section : position

	// The last option identifier used for each section.
	lastIdOption map[string]int // Section : last identifier

	// Section -> option : value
	data map[string]map[string]*tValue
}
~~~

`tValue` hold the input position for a value.

~~~ {prettyprint}
type tValue struct {
	position int    // Option order
	v        string // value
}
~~~

So all the configurations is formed up a tree with these two structures.

![config_tree](/public/images/read_revel/config_tree.png)

### Section

The `Config` supports four method for managing section. 

#### Method - `AddSection`

This method is used to add a new section to the configuration.

~~~ {prettyprint}
// AddSection adds a new section to the configuration.
//
// If the section is nil then uses the section by default which it's already
// created.
//
// It returns true if the new section was inserted, and false if the section
// already existed.
func (self *Config) AddSection(section string) bool {
	// DEFAULT_SECTION
	if section == "" {
		return false
	}

	if _, ok := self.data[section]; ok {
		return false
	}

	self.data[section] = make(map[string]*tValue)

	// Section order
	self.idSection[section] = self.lastIdSection
	self.lastIdSection++

	return true
}
~~~
- `lastIdSection` logs the next position for adding new section.

#### Method - `RemoveSection`

This method is used to remove a section from the configuration.

~~~ {prettyprint}
// RemoveSection removes a section from the configuration.
// It returns true if the section was removed, and false if section did not exist.
func (self *Config) RemoveSection(section string) bool {
	_, ok := self.data[section]

	// Default section cannot be removed.
	if !ok || section == DEFAULT_SECTION {
		return false
	}

	for o, _ := range self.data[section] {
		delete(self.data[section], o) // *value
	}
	delete(self.data, section)

	delete(self.lastIdOption, section)
	delete(self.idSection, section)

	return true
}
~~~
- Can't remove default section

#### Method - `HasSection`

This method is used to check the specified section is exist in the configuration.

~~~ {prettyprint}
// HasSection checks if the configuration has the given section.
// (The default section always exists.)
func (self *Config) HasSection(section string) bool {
	_, ok := self.data[section]

	return ok
}
~~~

#### Method - `Sections`

This method is to get all the existing sections' names.

~~~ {prettyprint}
// Sections returns the list of sections in the configuration.
// (The default section always exists.)
func (self *Config) Sections() (sections []string) {
	sections = make([]string, len(self.idSection))
	pos := 0 // Position in sections

	for i := 0; i < self.lastIdSection; i++ {
		for section, id := range self.idSection {
			if id == i {
				sections[pos] = section
				pos++
			}
		}
	}

	return sections
}
~~~

### Option

The option is represented by the `tValue`. And it is almost same as the section.

### Type

The value of a option is saved as string. `goconfig` support following types base on the raw string.

- `Bool`: convert the response string to bool.
- `Float`: convert the response string to Float.
- `Int`: convert the response string to Int.
- `RawString`: get raw string from the specified section, if not found, find it in default section.
- `RawStringDefault`: get raw string only from default section.
- `String`: get raw string from the specified section, unfolding automatically if need.

### Read/Write file

The configuration also support getting informations from local file and saving informations into a
local file.

## MergedConfig

Revel wrap the `goconfig` in its own structure, named `MergedConfig`

~~~ {prettyprint}
// It has a "preferred" section that is checked first for option queries.
// If the preferred section does not have the option, the DEFAULT section is
// checked fallback.
type MergedConfig struct {
	config  *config.Config
	section string // Check this section first, then fall back to DEFAULT
}
~~~
As the comment says, revel first search the section specified by the `section` field.

The wrapped methods handle all the errors in searching, and reported it by the boolean variable.

One special method is `MergedConfig`'s Options, it accepts a filter prefix string. The results from
the config.Options will be filtered.

~~~ {prettyprint}
// Options returns all configuration option keys.
// If a prefix is provided, then that is applied as a filter.
func (c *MergedConfig) Options(prefix string) []string {
	var options []string
	keys, _ := c.config.Options(c.section)
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			options = append(options, key)
		}
	}
	return options
}
~~~

FIN.
