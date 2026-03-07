//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type requestPreset struct {
	Name   string
	Method string
	Path   string
	Query  string
	Body   string
}

type responsePayload struct {
	StatusCode int
	Status     string
	Headers    map[string]string
	RawBody    string
	JSONBody   any
}

type dynamicTableModel struct {
	walk.TableModelBase
	headers []string
	rows    [][]string
}

func newDynamicTableModel(headers []string) *dynamicTableModel {
	return &dynamicTableModel{headers: append([]string(nil), headers...)}
}

func (m *dynamicTableModel) RowCount() int {
	return len(m.rows)
}

func (m *dynamicTableModel) Value(row, col int) interface{} {
	if row < 0 || row >= len(m.rows) || col < 0 || col >= len(m.rows[row]) {
		return ""
	}
	return m.rows[row][col]
}

func (m *dynamicTableModel) SetData(headers []string, rows [][]string) {
	m.headers = append([]string(nil), headers...)
	m.rows = append([][]string(nil), rows...)
	m.PublishRowsReset()
}

type desktopApp struct {
	client *http.Client

	mainWindow *walk.MainWindow
	baseURL    *walk.LineEdit
	method     *walk.ComboBox
	path       *walk.LineEdit
	query      *walk.LineEdit
	body       *walk.TextEdit
	status     *walk.TextLabel
	jsonView   *walk.TextEdit
	headers    *walk.TextEdit
	overview   *walk.TextLabel
	presets    *walk.ListBox

	responseTable   *walk.TableView
	categoriesTable *walk.TableView
	productsTable   *walk.TableView
	schemaTable     *walk.TableView

	responseModel   *dynamicTableModel
	categoriesModel *dynamicTableModel
	productsModel   *dynamicTableModel
	schemaModel     *dynamicTableModel

	methods []string
	preset  []requestPreset
}

func main() {
	app := &desktopApp{
		client: &http.Client{Timeout: 15 * time.Second},
		methods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
		},
		preset: buildPresets(),
	}

	if err := app.buildUI(); err != nil {
		panic(fmt.Errorf("build ui: %w", err))
	}

	app.applyPreset(0)
	app.refreshOverview()
	app.mainWindow.Run()
}

func (a *desktopApp) buildUI() error {
	presetNames := make([]string, 0, len(a.preset))
	for _, p := range a.preset {
		presetNames = append(presetNames, fmt.Sprintf("%s  %s", p.Method, p.Name))
	}

	a.responseModel = newDynamicTableModel([]string{"Info"})
	a.categoriesModel = newDynamicTableModel([]string{"Info"})
	a.productsModel = newDynamicTableModel([]string{"Info"})
	a.schemaModel = newDynamicTableModel([]string{"Resource", "Column", "Type", "Access", "Description"})

	err := (MainWindow{
		AssignTo: &a.mainWindow,
		Title:    "Lab 2 API Desktop Client",
		Size:     Size{1450, 920},
		Layout:   VBox{MarginsZero: true, SpacingZero: true},
		Children: []Widget{
			Composite{
				Layout: HBox{Margins: Margins{Left: 10, Top: 8, Right: 10, Bottom: 8}},
				Children: []Widget{
					TextLabel{
						Text: "Lab 2 API Desktop Client",
						Font: Font{Family: "Segoe UI", PointSize: 11, Bold: true},
					},
					HSpacer{},
					TextLabel{Text: "Local GUI app. No Docker and no extra port for the client."},
				},
			},
			HSplitter{
				Children: []Widget{
					Composite{
						Layout: VBox{Margins: Margins{Left: 10, Top: 6, Right: 10, Bottom: 10}},
						Children: []Widget{
							Label{Text: "API Base URL"},
							LineEdit{
								AssignTo: &a.baseURL,
								Text:     getEnv("API_BASE_URL", "http://localhost:4200"),
							},
							PushButton{
								Text: "Refresh Overview",
								OnClicked: func() {
									a.refreshOverview()
								},
							},
							Label{
								Text: "Saved Requests",
								Font: Font{Family: "Segoe UI", PointSize: 9, Bold: true},
							},
							ListBox{
								AssignTo: &a.presets,
								Model:    presetNames,
								OnCurrentIndexChanged: func() {
									a.applyPreset(a.presets.CurrentIndex())
								},
							},
							TextLabel{
								Text: "Only /categories and /products are allowed in this client.",
							},
						},
					},
					Composite{
						Layout: VBox{Margins: Margins{Left: 0, Top: 6, Right: 10, Bottom: 10}},
						Children: []Widget{
							GroupBox{
								Title:  "Request Composer",
								Layout: VBox{Margins: Margins{Left: 8, Top: 10, Right: 8, Bottom: 8}},
								Children: []Widget{
									Composite{
										Layout: Grid{Columns: 2},
										Children: []Widget{
											Label{Text: "Method"},
											ComboBox{
												AssignTo:     &a.method,
												Model:        a.methods,
												Editable:     false,
												CurrentIndex: 0,
											},
											Label{Text: "Path"},
											LineEdit{
												AssignTo: &a.path,
												Text:     "/categories",
											},
											Label{Text: "Query"},
											LineEdit{
												AssignTo: &a.query,
												Text:     "page=1&limit=10",
											},
										},
									},
									Label{Text: "JSON Body"},
									TextEdit{
										AssignTo: &a.body,
										VScroll:  true,
									},
									Composite{
										Layout: HBox{Spacing: 8},
										Children: []Widget{
											PushButton{
												Text: "Send Request",
												OnClicked: func() {
													a.sendRequest()
												},
											},
											PushButton{
												Text: "Format JSON",
												OnClicked: func() {
													a.formatBody()
												},
											},
											HSpacer{},
											TextLabel{
												AssignTo: &a.status,
												Text:     "Ready",
											},
										},
									},
								},
							},
							TabWidget{
								Pages: []TabPage{
									{
										Title:  "Response",
										Layout: VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}},
										Children: []Widget{
											HSplitter{
												Children: []Widget{
													TextEdit{
														AssignTo: &a.jsonView,
														ReadOnly: true,
														VScroll:  true,
														Text:     "{\r\n  \"hint\": \"Choose a preset and send a request.\"\r\n}",
													},
													TextEdit{
														AssignTo: &a.headers,
														ReadOnly: true,
														VScroll:  true,
														Text:     "{}",
													},
												},
											},
											TableView{
												AssignTo:            &a.responseTable,
												AlternatingRowBG:    true,
												LastColumnStretched: true,
												Model:               a.responseModel,
											},
										},
									},
									{
										Title:  "Overview",
										Layout: VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}},
										Children: []Widget{
											TextLabel{
												AssignTo: &a.overview,
												Text:     "Categories: 0 | Products: 0",
											},
											HSplitter{
												Children: []Widget{
													TableView{
														AssignTo:            &a.categoriesTable,
														AlternatingRowBG:    true,
														LastColumnStretched: true,
														Model:               a.categoriesModel,
													},
													TableView{
														AssignTo:            &a.productsTable,
														AlternatingRowBG:    true,
														LastColumnStretched: true,
														Model:               a.productsModel,
													},
												},
											},
										},
									},
									{
										Title:  "Schema",
										Layout: VBox{Margins: Margins{Left: 8, Top: 8, Right: 8, Bottom: 8}},
										Children: []Widget{
											TableView{
												AssignTo:            &a.schemaTable,
												AlternatingRowBG:    true,
												LastColumnStretched: true,
												Model:               a.schemaModel,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}).Create()

	if err != nil {
		return err
	}

	if err := a.setTableColumns(a.responseTable, []string{"Info"}); err != nil {
		return err
	}
	if err := a.setTableColumns(a.categoriesTable, []string{"Info"}); err != nil {
		return err
	}
	if err := a.setTableColumns(a.productsTable, []string{"Info"}); err != nil {
		return err
	}
	if err := a.setTableColumns(a.schemaTable, a.schemaModel.headers); err != nil {
		return err
	}
	a.schemaModel.SetData(a.schemaModel.headers, buildSchemaRows())

	return nil
}

func (a *desktopApp) applyPreset(index int) {
	if index < 0 || index >= len(a.preset) {
		return
	}
	p := a.preset[index]
	for i, method := range a.methods {
		if method == p.Method {
			_ = a.method.SetCurrentIndex(i)
			break
		}
	}
	a.path.SetText(p.Path)
	a.query.SetText(p.Query)
	a.body.SetText(p.Body)
	a.setStatus(fmt.Sprintf("Loaded preset: %s", p.Name))
}

func (a *desktopApp) sendRequest() {
	method := a.selectedMethod()
	path := strings.TrimSpace(a.path.Text())
	query := strings.TrimSpace(a.query.Text())
	body := strings.TrimSpace(a.body.Text())

	if !isAllowedMethod(method) {
		a.setStatus("Unsupported method")
		return
	}
	if !isAllowedPath(path) {
		a.setStatus("Path must start with /categories or /products")
		return
	}

	targetURL, err := composeTargetURL(strings.TrimSpace(a.baseURL.Text()), path, query)
	if err != nil {
		a.setStatus(fmt.Sprintf("URL error: %v", err))
		return
	}

	a.setStatus("Sending request...")
	start := time.Now()
	resp, err := a.doRequest(method, targetURL, body)
	if err != nil {
		a.setStatus(fmt.Sprintf("Request failed: %v", err))
		return
	}
	a.renderResponse(resp, time.Since(start))

	if method != http.MethodGet {
		a.refreshOverview()
	}
}

func (a *desktopApp) formatBody() {
	raw := strings.TrimSpace(a.body.Text())
	if raw == "" {
		return
	}

	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		a.setStatus(fmt.Sprintf("JSON format error: %v", err))
		return
	}

	pretty, _ := json.MarshalIndent(parsed, "", "  ")
	a.body.SetText(string(pretty))
	a.setStatus("Body formatted")
}

func (a *desktopApp) refreshOverview() {
	baseURL := strings.TrimRight(strings.TrimSpace(a.baseURL.Text()), "/")
	categories, err := a.doRequest(http.MethodGet, baseURL+"/categories?limit=100", "")
	if err != nil {
		_ = a.setTableData(a.categoriesTable, a.categoriesModel, []string{"Error"}, [][]string{{err.Error()}})
		a.setStatus(fmt.Sprintf("Overview error: %v", err))
		return
	}

	products, err := a.doRequest(http.MethodGet, baseURL+"/products?limit=100", "")
	if err != nil {
		_ = a.setTableData(a.productsTable, a.productsModel, []string{"Error"}, [][]string{{err.Error()}})
		a.setStatus(fmt.Sprintf("Overview error: %v", err))
		return
	}

	categoryRows := extractRows(categories.JSONBody)
	productRows := extractRows(products.JSONBody)

	_ = a.setTableData(a.categoriesTable, a.categoriesModel, []string{
		"id", "name", "description", "status", "createdAt",
	}, rowsForHeaders(categoryRows, []string{
		"id", "name", "description", "status", "createdAt",
	}))

	_ = a.setTableData(a.productsTable, a.productsModel, []string{
		"id", "categoryId", "categoryName", "name", "description", "price", "status", "createdAt",
	}, rowsForHeaders(productRows, []string{
		"id", "categoryId", "categoryName", "name", "description", "price", "status", "createdAt",
	}))

	a.overview.SetText(fmt.Sprintf("Categories: %d | Products: %d", len(categoryRows), len(productRows)))
	a.setStatus("Overview refreshed")
}

func (a *desktopApp) renderResponse(resp responsePayload, duration time.Duration) {
	a.setStatus(fmt.Sprintf("%s in %d ms", resp.Status, duration.Milliseconds()))

	if resp.JSONBody != nil {
		pretty, _ := json.MarshalIndent(resp.JSONBody, "", "  ")
		a.jsonView.SetText(string(pretty))
	} else if resp.RawBody != "" {
		a.jsonView.SetText(resp.RawBody)
	} else {
		a.jsonView.SetText("{}")
	}

	if len(resp.Headers) > 0 {
		headersPretty, _ := json.MarshalIndent(resp.Headers, "", "  ")
		a.headers.SetText(string(headersPretty))
	} else {
		a.headers.SetText("{}")
	}

	responseRows := extractRows(resp.JSONBody)
	if len(responseRows) == 0 {
		_ = a.setTableData(a.responseTable, a.responseModel, []string{"Info"}, [][]string{{"No tabular rows in response."}})
		return
	}

	headers := collectHeaders(responseRows)
	_ = a.setTableData(a.responseTable, a.responseModel, headers, rowsForHeaders(responseRows, headers))
}

func (a *desktopApp) setTableData(tv *walk.TableView, model *dynamicTableModel, headers []string, rows [][]string) error {
	if len(headers) == 0 {
		headers = []string{"Info"}
	}
	if len(rows) == 0 {
		rows = [][]string{{"No rows available."}}
	}

	if err := a.setTableColumns(tv, headers); err != nil {
		return err
	}
	model.SetData(headers, rows)
	return nil
}

func (a *desktopApp) setTableColumns(tv *walk.TableView, headers []string) error {
	columns := tv.Columns()
	if err := columns.Clear(); err != nil {
		return err
	}

	for _, header := range headers {
		col := walk.NewTableViewColumn()
		col.SetTitle(header)
		width := 140
		if len(header) > 14 {
			width = 220
		}
		col.SetWidth(width)
		if err := columns.Add(col); err != nil {
			return err
		}
	}
	return nil
}

func (a *desktopApp) doRequest(method, targetURL, rawBody string) (responsePayload, error) {
	var bodyReader io.Reader
	if rawBody != "" {
		bodyReader = bytes.NewBufferString(rawBody)
	}

	req, err := http.NewRequest(method, targetURL, bodyReader)
	if err != nil {
		return responsePayload{}, err
	}

	req.Header.Set("Accept", "application/json")
	if rawBody != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return responsePayload{}, err
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return responsePayload{}, err
	}

	headers := make(map[string]string)
	for key, values := range resp.Header {
		headers[key] = strings.Join(values, ", ")
	}

	result := responsePayload{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Headers:    headers,
		RawBody:    string(payload),
	}

	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") && len(payload) > 0 {
		var parsed any
		if err := json.Unmarshal(payload, &parsed); err == nil {
			result.JSONBody = parsed
		}
	}

	return result, nil
}

func (a *desktopApp) selectedMethod() string {
	index := a.method.CurrentIndex()
	if index >= 0 && index < len(a.methods) {
		return a.methods[index]
	}
	return strings.ToUpper(strings.TrimSpace(a.method.Text()))
}

func (a *desktopApp) setStatus(message string) {
	a.status.SetText(message)
}

func buildPresets() []requestPreset {
	return []requestPreset{
		{Name: "List categories", Method: "GET", Path: "/categories", Query: "page=1&limit=10"},
		{Name: "Create category", Method: "POST", Path: "/categories", Body: "{\n  \"name\": \"Face Care\",\n  \"description\": \"Creams and serums\",\n  \"status\": \"active\"\n}"},
		{Name: "Update category", Method: "PUT", Path: "/categories/<category-id>", Body: "{\n  \"name\": \"Face Care Pro\",\n  \"description\": \"Updated description\",\n  \"status\": \"hidden\"\n}"},
		{Name: "List products", Method: "GET", Path: "/products", Query: "page=1&limit=10"},
		{Name: "Create product", Method: "POST", Path: "/products", Body: "{\n  \"categoryId\": \"<category-id>\",\n  \"name\": \"Hydrating Cream\",\n  \"description\": \"50 ml\",\n  \"price\": 990.5,\n  \"status\": \"available\"\n}"},
		{Name: "Patch product", Method: "PATCH", Path: "/products/<product-id>", Body: "{\n  \"status\": \"out_of_stock\"\n}"},
		{Name: "Delete product", Method: "DELETE", Path: "/products/<product-id>"},
	}
}

func buildSchemaRows() [][]string {
	return [][]string{
		{"categories", "id", "uuid", "read", "Primary key returned by the API."},
		{"categories", "name", "varchar", "write", "Required category title."},
		{"categories", "description", "text", "write", "Optional description."},
		{"categories", "status", "varchar", "write", "Allowed values: active, hidden."},
		{"categories", "created_at", "timestamp", "read", "Creation timestamp."},
		{"categories", "updated_at", "timestamp", "read", "Last update timestamp."},
		{"categories", "deleted_at", "timestamp", "soft-delete", "Used for hidden rows after DELETE."},
		{"products", "id", "uuid", "read", "Primary key returned by the API."},
		{"products", "category_id", "uuid", "write", "Foreign key to categories."},
		{"products", "name", "varchar", "write", "Required product title."},
		{"products", "description", "text", "write", "Optional description."},
		{"products", "price", "decimal(10,2)", "write", "Price must be non-negative."},
		{"products", "status", "varchar", "write", "Allowed values: available, out_of_stock, discontinued."},
		{"products", "stock_quantity", "integer", "db-only", "Present in SQL migration, not exposed by DTOs."},
		{"products", "created_at", "timestamp", "read", "Creation timestamp."},
		{"products", "updated_at", "timestamp", "read", "Last update timestamp."},
		{"products", "deleted_at", "timestamp", "soft-delete", "Used for hidden rows after DELETE."},
	}
}

func extractRows(payload any) []map[string]any {
	switch value := payload.(type) {
	case map[string]any:
		if data, ok := value["data"].([]any); ok {
			return sliceToMapRows(data)
		}
		return []map[string]any{value}
	case []any:
		return sliceToMapRows(value)
	default:
		return nil
	}
}

func sliceToMapRows(items []any) []map[string]any {
	rows := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if row, ok := item.(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func rowsForHeaders(items []map[string]any, headers []string) [][]string {
	if len(items) == 0 {
		return [][]string{{"No rows available."}}
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		line := make([]string, 0, len(headers))
		for _, header := range headers {
			line = append(line, stringifyCell(item[header]))
		}
		rows = append(rows, line)
	}
	return rows
}

func collectHeaders(rows []map[string]any) []string {
	set := make(map[string]struct{})
	for _, row := range rows {
		for key := range row {
			set[key] = struct{}{}
		}
	}

	headers := make([]string, 0, len(set))
	for key := range set {
		headers = append(headers, key)
	}
	sort.Strings(headers)
	return headers
}

func stringifyCell(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(encoded)
	}
}

func composeTargetURL(baseURL, path, query string) (string, error) {
	if baseURL == "" {
		return "", fmt.Errorf("base URL is empty")
	}

	target, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return "", err
	}

	if query != "" {
		values, err := url.ParseQuery(query)
		if err != nil {
			return "", err
		}
		target.RawQuery = values.Encode()
	}

	return target.String(), nil
}

func isAllowedMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isAllowedPath(path string) bool {
	if !strings.HasPrefix(path, "/") {
		return false
	}
	return strings.HasPrefix(path, "/categories") || strings.HasPrefix(path, "/products")
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
