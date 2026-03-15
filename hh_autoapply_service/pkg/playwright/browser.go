package playwright

import (
	"encoding/json"
	"fmt"
	"sync"

	playwright "github.com/playwright-community/playwright-go"
)

type BrowserManager struct {
	pw       playwright.Playwright
	browser  playwright.Browser
	mu       sync.RWMutex
	headless bool
	slowMo   int
}

type BrowserPage struct {
	Page    playwright.Page
	Context playwright.BrowserContext
}

// StorageState структура для восстановления сессии
type StorageState struct {
	Cookies []playwright.Cookie `json:"cookies"`
	Origins []OriginStorage     `json:"origins"`
}

type OriginStorage struct {
	Origin       string         `json:"origin"`
	LocalStorage []LocalStorage `json:"localStorage"`
}

type LocalStorage struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func NewBrowserManager(headless bool, slowMo int) (*BrowserManager, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start playwright: %w", err)
	}

	browser, err := pw.Firefox.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		SlowMo:   playwright.Float(float64(slowMo)),
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	return &BrowserManager{
		pw:       *pw,
		browser:  browser,
		headless: headless,
		slowMo:   slowMo,
	}, nil
}

func (m *BrowserManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.browser != nil {
		m.browser.Close()
	}
	m.pw.Stop()
}

func (m *BrowserManager) NewPage(storageState string) (*BrowserPage, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.browser == nil {
		return nil, fmt.Errorf("browser is not initialized")
	}

	contextOptions := playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	// Если есть storageState, используем его для восстановления сессии
	if storageState != "" {
		// Сохраняем storageState во временный файл и используем SetStorageState
		// Playwright-go не поддерживает прямую загрузку из строки
		// Поэтому просто создаем контекст без storageState
		// А cookies добавим отдельно через AddCookies
	}

	page, err := m.browser.NewContext(contextOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}

	// Если есть storageState, восстанавливаем cookies
	if storageState != "" {
		var state StorageState
		if err := json.Unmarshal([]byte(storageState), &state); err != nil {
			page.Close()
			return nil, fmt.Errorf("failed to parse storage state: %w", err)
		}

		// Добавляем cookies
		if len(state.Cookies) > 0 {
			optionalCookies := make([]playwright.OptionalCookie, len(state.Cookies))
			for i, cookie := range state.Cookies {
				c := cookie
				optionalCookies[i] = playwright.OptionalCookie{
					Name:     c.Name,
					Value:    c.Value,
					Domain:   &c.Domain,
					Path:     &c.Path,
					Secure:   &c.Secure,
					HttpOnly: &c.HttpOnly,
				}
			}
			if err := page.AddCookies(optionalCookies); err != nil {
				page.Close()
				return nil, fmt.Errorf("failed to add cookies: %w", err)
			}
		}
	}

	newPage, err := page.NewPage()
	if err != nil {
		page.Close()
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	// Скрипт для скрытия автоматизации
	script := playwright.Script{
		Content: playwright.String(`() => {
			Object.defineProperty(navigator, 'webdriver', { get: () => undefined });
			Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3, 4, 5] });
			Object.defineProperty(navigator, 'languages', { get: () => ['ru-RU', 'ru', 'en-US', 'en'] });
		}`),
	}
	newPage.AddInitScript(script)

	return &BrowserPage{
		Page:    newPage,
		Context: page,
	}, nil
}

// NewPageWithToken создает страницу с токеном HH.ru (storageState JSON)
func (m *BrowserManager) NewPageWithToken(storageState string) (*BrowserPage, error) {
	return m.NewPage(storageState)
}

func (bp *BrowserPage) Close() {
	if bp.Context != nil {
		bp.Context.Close()
	}
}
