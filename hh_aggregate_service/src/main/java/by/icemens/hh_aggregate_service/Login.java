package by.icemens.hh_aggregate_service;

import by.icemens.hh_aggregate_service.config.Settings;
import com.microsoft.playwright.*;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.nio.file.Files;
import java.util.Scanner;

/**
 * Класс для авторизации на HH.ru.
 * Запустите main() для входа и сохранения сессии.
 */
public class Login {

    private static final Logger logger = LoggerFactory.getLogger(Login.class);

    public static void main(String[] args) {
        Settings settings = new Settings();

        logger.info("============================================================");
        logger.info("           Авторизация на HH.ru");
        logger.info("============================================================");

        Playwright playwright = null;
        Browser browser = null;
        BrowserContext context = null;

        try {
            logger.info("[1] Запуск Playwright...");
            playwright = Playwright.create();

            logger.info("[2] Запуск браузера Chromium...");
            browser = playwright.chromium().launch(new BrowserType.LaunchOptions()
                .setHeadless(false)  // Показываем браузер
                .setSlowMo(100));    // Замедление для наглядности

            logger.info("[3] Создание контекста браузера...");
            context = browser.newContext();

            Page page = context.newPage();

            // Устанавливаем реалистичный User-Agent
            page.setExtraHTTPHeaders(java.util.Map.of(
                "User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
            ));

            logger.info("[4] Переход на страницу входа HH.ru...");
            logger.info("    URL: https://hh.ru/login");

            // Переход на страницу
            page.navigate("https://hh.ru/login", new Page.NavigateOptions().setTimeout(30000));

            logger.info("[5] Страница загружена!");
            logger.info("    Заголовок: {}", page.title());
            logger.info("    URL: {}", page.url());

            logger.info("------------------------------------------------------------");
            logger.info("ДЕЙСТВИЯ:");
            logger.info("1. Войдите в свой аккаунт HH.ru в открытом окне браузера");
            logger.info("2. Дождитесь загрузки личного кабинета (должна появиться ваша");
            logger.info("   фамилия в правом верхнем углу)");
            logger.info("3. Вернитесь сюда и нажмите Enter");
            logger.info("------------------------------------------------------------");
            logger.info("Нажмите Enter после успешного входа... ");

            new Scanner(System.in).nextLine();

            // Проверка наличия токена
            logger.info("[6] Проверка куки...");
            var cookies = context.cookies();
            boolean hasToken = cookies.stream()
                .anyMatch(c -> c.name.equals("hhtoken"));

            if (!hasToken) {
                logger.warn("'hhtoken' не найден в куки!");
                logger.warn("Возможно, вы ещё не вошли в аккаунт.");
                logger.info("Доступные куки:");
                for (var cookie : cookies) {
                    logger.info("  - {}", cookie.name);
                }
                logger.info("Продолжить и сохранить сессию? (y/n): ");
                String response = new Scanner(System.in).nextLine().trim().toLowerCase();
                if (!response.equals("y") && !response.equals("yes")) {
                    logger.info("Авторизация отменена.");
                    return;
                }
            } else {
                logger.info("[OK] 'hhtoken' найден! Аутентификация успешна.");
            }

            // Сохранение сессии
            logger.info("[7] Сохранение сессии...");
            context.storageState(new BrowserContext.StorageStateOptions()
                .setPath(settings.getSessionFile()));

            logger.info("[OK] Сессия сохранена в: {}", settings.getSessionFile().toAbsolutePath());

            // Проверка файла
            if (Files.exists(settings.getSessionFile())) {
                long size = Files.size(settings.getSessionFile());
                logger.info("[OK] Размер файла: {} байт", size);
            }

        } catch (Exception e) {
            logger.error("Ошибка: {}", e.getMessage(), e);
            System.exit(1);
        } finally {
            // Закрываем браузер
            if (context != null) context.close();
            if (browser != null) browser.close();
            if (playwright != null) playwright.close();
        }

        logger.info("============================================================");
        logger.info("Готово! Теперь можно запускать парсинг вакансий:");
        logger.info("  gradle run");
        logger.info("============================================================");
    }
}
