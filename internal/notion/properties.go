package notion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jomei/notionapi"
)

// dateLayout is the form in which dates are shown and typed.
const dateLayout = "2006-01-02"

// PropertyValue is one page property prepared for display.
type PropertyValue struct {
	Name string
	Type notionapi.PropertyType
	// Text is the value as shown to the user; empty when the property is unset.
	Text string
	// Checked is the value of a checkbox property.
	Checked bool
}

// Editable reports whether the app can change this property.
func (p PropertyValue) Editable() bool {
	switch p.Type {
	case notionapi.PropertyTypeStatus,
		notionapi.PropertyTypeSelect,
		notionapi.PropertyTypeCheckbox,
		notionapi.PropertyTypeDate:
		return true
	default:
		return false
	}
}

// PageProperties returns a page's properties except its title, ordered so
// the ones people act on come first (status, select, date, checkbox) and the
// rest follow alphabetically.
func PageProperties(page *notionapi.Page) []PropertyValue {
	if page == nil {
		return nil
	}

	values := make([]PropertyValue, 0, len(page.Properties))
	for name, property := range page.Properties {
		if property == nil || property.GetType() == notionapi.PropertyTypeTitle {
			continue
		}
		value := PropertyValue{Name: name, Type: property.GetType()}
		value.Text, value.Checked = propertyText(property)
		values = append(values, value)
	}

	sort.Slice(values, func(i, j int) bool {
		ri, rj := propertyRank(values[i].Type), propertyRank(values[j].Type)
		if ri != rj {
			return ri < rj
		}
		return values[i].Name < values[j].Name
	})
	return values
}

// propertyRank orders property types for display.
func propertyRank(propertyType notionapi.PropertyType) int {
	switch propertyType {
	case notionapi.PropertyTypeStatus:
		return 0
	case notionapi.PropertyTypeSelect:
		return 1
	case notionapi.PropertyTypeDate:
		return 2
	case notionapi.PropertyTypeCheckbox:
		return 3
	default:
		return 4
	}
}

// propertyText renders a property value as text. The second result is the
// state of a checkbox property.
func propertyText(property notionapi.Property) (text string, checked bool) {
	switch p := property.(type) {
	case *notionapi.RichTextProperty:
		return GetRichTextString(p.RichText), false
	case *notionapi.NumberProperty:
		// The library decodes an unset number as 0, so the two cannot be
		// told apart. Showing nothing for 0 avoids a "0" on every row that
		// has no number, at the cost of hiding a real zero.
		if p.Number == 0 {
			return "", false
		}
		return strconv.FormatFloat(p.Number, 'f', -1, 64), false
	case *notionapi.SelectProperty:
		return p.Select.Name, false
	case *notionapi.StatusProperty:
		return p.Status.Name, false
	case *notionapi.MultiSelectProperty:
		names := make([]string, 0, len(p.MultiSelect))
		for _, option := range p.MultiSelect {
			names = append(names, option.Name)
		}
		return strings.Join(names, ", "), false
	case *notionapi.DateProperty:
		return formatDateRange(p.Date), false
	case *notionapi.CheckboxProperty:
		if p.Checkbox {
			return "yes", true
		}
		return "no", false
	case *notionapi.URLProperty:
		return p.URL, false
	case *notionapi.EmailProperty:
		return p.Email, false
	case *notionapi.PhoneNumberProperty:
		return p.PhoneNumber, false
	case *notionapi.PeopleProperty:
		names := make([]string, 0, len(p.People))
		for _, person := range p.People {
			if person.Name != "" {
				names = append(names, person.Name)
			}
		}
		return strings.Join(names, ", "), false
	case *notionapi.CreatedTimeProperty:
		return formatTime(p.CreatedTime), false
	case *notionapi.LastEditedTimeProperty:
		return formatTime(p.LastEditedTime), false
	case *notionapi.UniqueIDProperty:
		if p.UniqueID.Prefix != nil && *p.UniqueID.Prefix != "" {
			return fmt.Sprintf("%s-%d", *p.UniqueID.Prefix, p.UniqueID.Number), false
		}
		return strconv.Itoa(p.UniqueID.Number), false
	case *notionapi.RelationProperty:
		if len(p.Relation) == 0 {
			return "", false
		}
		return fmt.Sprintf("%d linked", len(p.Relation)), false
	case *notionapi.FilesProperty:
		if len(p.Files) == 0 {
			return "", false
		}
		return fmt.Sprintf("%d file(s)", len(p.Files)), false
	case *notionapi.FormulaProperty:
		return formulaText(p.Formula), false
	default:
		return "", false
	}
}

func formulaText(formula notionapi.Formula) string {
	switch formula.Type {
	case notionapi.FormulaTypeString:
		return formula.String
	case notionapi.FormulaTypeNumber:
		return strconv.FormatFloat(formula.Number, 'f', -1, 64)
	case notionapi.FormulaTypeBoolean:
		if formula.Boolean {
			return "yes"
		}
		return "no"
	case notionapi.FormulaTypeDate:
		return formatDateRange(formula.Date)
	default:
		return ""
	}
}

// formatDateRange renders a date property: "2026-11-12", with a time when it
// has one, and "start → end" for ranges.
func formatDateRange(date *notionapi.DateObject) string {
	if date == nil || date.Start == nil {
		return ""
	}
	text := formatDate(time.Time(*date.Start))
	if date.End != nil {
		text += " → " + formatDate(time.Time(*date.End))
	}
	return text
}

func formatDate(t time.Time) string {
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
		return t.Format(dateLayout)
	}
	return t.Format(dateLayout + " 15:04")
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(dateLayout + " 15:04")
}

// PropertyOptions returns the choices a status or select property offers,
// in the order the database defines them.
func PropertyOptions(database *notionapi.Database, name string) []string {
	if database == nil {
		return nil
	}
	var options []notionapi.Option
	switch config := database.Properties[name].(type) {
	case *notionapi.StatusPropertyConfig:
		options = config.Status.Options
	case *notionapi.SelectPropertyConfig:
		options = config.Select.Options
	}
	names := make([]string, 0, len(options))
	for _, option := range options {
		names = append(names, option.Name)
	}
	return names
}

// PropertyUpdate is a new value for one page property.
type PropertyUpdate struct {
	Name  string
	value any
}

// StatusUpdate sets a status property to one of its options.
func StatusUpdate(name, option string) PropertyUpdate {
	return PropertyUpdate{Name: name, value: map[string]any{"status": map[string]any{"name": option}}}
}

// SelectUpdate sets a select property to one of its options; an empty option
// clears it.
func SelectUpdate(name, option string) PropertyUpdate {
	if option == "" {
		return PropertyUpdate{Name: name, value: map[string]any{"select": nil}}
	}
	return PropertyUpdate{Name: name, value: map[string]any{"select": map[string]any{"name": option}}}
}

// CheckboxUpdate sets a checkbox property.
func CheckboxUpdate(name string, checked bool) PropertyUpdate {
	return PropertyUpdate{Name: name, value: map[string]any{"checkbox": checked}}
}

// DateUpdate sets a date property from text typed by the user. Empty text
// clears the date. The date is sent without a time, so Notion shows a plain
// date rather than midnight.
func DateUpdate(name, text string) (PropertyUpdate, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return PropertyUpdate{Name: name, value: map[string]any{"date": nil}}, nil
	}
	if _, err := time.Parse(dateLayout, text); err != nil {
		return PropertyUpdate{}, fmt.Errorf("%q is not a date: use YYYY-MM-DD, for example %s", text, time.Now().Format(dateLayout))
	}
	return PropertyUpdate{Name: name, value: map[string]any{"date": map[string]any{"start": text}}}, nil
}

// GetDatabase retrieves a database, including the configuration of its
// properties (such as the options of a status property).
func (c *Client) GetDatabase(ctx context.Context, id string) (*notionapi.Database, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter wait: %w", err)
	}
	database, err := c.api.Database.Get(ctx, notionapi.DatabaseID(id))
	if err != nil {
		return nil, fmt.Errorf("get database %s: %w", id, err)
	}
	return database, nil
}

// SetPageProperty changes one property of a page.
func (c *Client) SetPageProperty(ctx context.Context, pageID string, update PropertyUpdate) error {
	if err := c.writable(); err != nil {
		return err
	}
	if update.Name == "" || update.value == nil {
		return errors.New("set page property: empty update")
	}
	body := map[string]any{"properties": map[string]any{update.Name: update.value}}
	path := "/v1/pages/" + url.PathEscape(pageID)
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, body, nil); err != nil {
		return fmt.Errorf("set %q on page %s: %w", update.Name, pageID, err)
	}
	return nil
}
