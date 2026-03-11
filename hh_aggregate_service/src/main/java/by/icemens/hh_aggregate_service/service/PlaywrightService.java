package by.icemens.hh_aggregate_service.service;

import by.icemens.hh_aggregate_service.config.PlaywrightConfig;
import com.microsoft.playwright.Browser;
import com.microsoft.playwright.BrowserContext;
import com.microsoft.playwright.Page;
import com.microsoft.playwright.Playwright;
import jakarta.annotation.PostConstruct;
import jakarta.annotation.PreDestroy;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Component;

import java.util.List;

import static by.icemens.hh_aggregate_service.config.PlaywrightConfig.DEFAULT_HEADERS;

@RequiredArgsConstructor
@Component
@Slf4j
public class PlaywrightService {

    private final PlaywrightConfig playwrightConfig;
    private Playwright playwright;
    private Browser browser;

    @PostConstruct
    public void init() {
        if (playwright == null) {
            log.info("Инициализация Playwright...");
            playwright = Playwright.create();
            browser = playwright.chromium().launch(
                    new com.microsoft.playwright.BrowserType.LaunchOptions()
                            .setHeadless(playwrightConfig.isHeadless())
                            .setSlowMo(100)
                            .setArgs(List.of(
                                    "--disable-blink-features=AutomationControlled",
                                    "--disable-dev-shm-usage",
                                    "--no-sandbox",
                                    "--disable-gpu"
                            ))
            );
            log.info("Браузер запущен");
        }
    }

    @PreDestroy
    public void destroy() {
        if (browser != null) {
            browser.close();
            browser = null;
        }
        if (playwright != null) {
            playwright.close();
            playwright = null;
        }
        log.info("Браузер остановлен");
    }

    /**
     * Получение страницы браузера с загрузкой сессии из бд
     */
    public BrowserPage getPage(Long userId) {
//        String hhToken = hhTokenService.getHhToken(userId).orElseThrow(
//                () -> new IllegalStateException(
//                "Файл сессии не найден. Авторизуйтесь в hh.ru."));
        return getPageFromStorage();
    }

    /**
     * Получение страницы браузера с загрузкой сессии
     */
    public BrowserPage getPageFromStorage() {
        if (browser == null) {
            init();
        }

        BrowserContext context = browser.newContext(
                new Browser.NewContextOptions()
//                        .setStorageState(storageState)
                        .setUserAgent(DEFAULT_HEADERS.get("User-Agent"))
        );

        Page page = context.newPage();
        page.setDefaultTimeout(30000);

        // Скрипт для скрытия факта автоматизации
        page.addInitScript("() => { " +
                "Object.defineProperty(navigator, 'webdriver', { get: () => undefined }); " +
                "Object.defineProperty(navigator, 'plugins', { get: () => [1, 2, 3, 4, 5] }); " +
                "Object.defineProperty(navigator, 'languages', { get: () => ['ru-RU', 'ru', 'en-US', 'en'] }); " +
                "}");

        return new BrowserPage(page, context);
    }

    public static class BrowserPage implements AutoCloseable {
        private final Page page;
        private final BrowserContext context;

        public BrowserPage(Page page, BrowserContext context) {
            this.page = page;
            this.context = context;
        }

        public Page get() {
            return page;
        }

        @Override
        public void close() {
            context.close();
        }
    }
}
