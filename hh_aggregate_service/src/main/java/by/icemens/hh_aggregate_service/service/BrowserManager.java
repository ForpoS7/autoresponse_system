package by.icemens.hh_aggregate_service.service;

import by.icemens.hh_aggregate_service.config.Settings;
import com.microsoft.playwright.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.nio.file.Files;
import java.util.concurrent.locks.ReentrantLock;

/**
 * Менеджер браузера Playwright.
 */
public class BrowserManager implements AutoCloseable {
    
    private static final Logger logger = LoggerFactory.getLogger(BrowserManager.class);
    
    private final Settings settings;
    private Playwright playwright;
    private Browser browser;
    private final ReentrantLock lock = new ReentrantLock();
    
    public BrowserManager(Settings settings) {
        this.settings = settings;
    }
    
    /**
     * Инициализация Playwright и запуск браузера.
     */
    public void start() {
        lock.lock();
        try {
            if (playwright == null) {
                logger.info("Запуск Playwright...");
                playwright = Playwright.create();
                // Запускаем браузер с дополнительными аргументами для обхода детекции
                browser = playwright.chromium().launch(new BrowserType.LaunchOptions()
                    .setHeadless(false)  // Всегда показываем браузер для отладки
                    .setArgs(java.util.List.of(
                        "--disable-blink-features=AutomationControlled",
                        "--disable-dev-shm-usage",
                        "--no-sandbox"
                    )));
                logger.info("Браузер запущен");
            }
        } finally {
            lock.unlock();
        }
    }
    
    /**
     * Остановка браузера и Playwright.
     */
    public void stop() {
        lock.lock();
        try {
            if (browser != null) {
                browser.close();
                browser = null;
            }
            if (playwright != null) {
                playwright.close();
                playwright = null;
            }
            logger.info("Браузер остановлен");
        } finally {
            lock.unlock();
        }
    }
    
    /**
     * Проверка наличия файла сессии.
     */
    private void validateSession() {
        if (!Files.exists(settings.getSessionFile())) {
            throw new IllegalStateException(
                "Файл сессии не найден: " + settings.getSessionFile() + 
                ". Запустите Login.main() для авторизации."
            );
        }
    }
    
    /**
     * Получение страницы браузера с сессией.
     */
    public BrowserPage getPage() {
        if (browser == null) {
            start();
        }

        validateSession();
        
        // Устанавливаем реалистичный User-Agent как в Login.java
        BrowserContext context = browser.newContext(new Browser.NewContextOptions()
            .setStorageStatePath(settings.getSessionFile())
            .setUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"));

        Page page = context.newPage();
        page.setDefaultTimeout(settings.getPageTimeout());
        
        // Скрываем признак автоматизации
        page.addInitScript("() => { Object.defineProperty(navigator, 'webdriver', { get: () => undefined }); }");

        return new BrowserPage(page, context);
    }
    
    @Override
    public void close() {
        stop();
    }
    
    /**
     * Обёртка над Page для автоматического закрытия контекста.
     */
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
    
    /**
     * Страница для интерактивного режима.
     */
    public static class InteractivePage implements AutoCloseable {
        public final Playwright playwright;
        public final Browser browser;
        public final BrowserContext context;
        public final Page page;
        
        public InteractivePage(Playwright playwright, Browser browser, 
                               BrowserContext context, Page page) {
            this.playwright = playwright;
            this.browser = browser;
            this.context = context;
            this.page = page;
        }
        
        @Override
        public void close() {
            browser.close();
            playwright.close();
        }
    }
}
