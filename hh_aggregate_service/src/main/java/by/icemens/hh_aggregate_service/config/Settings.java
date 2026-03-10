package by.icemens.hh_aggregate_service.config;

import lombok.Data;

import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;

/**
 * Простая конфигурация приложения.
 */
@Data
public class Settings {

    // Путь к директории для хранения сессии
    private Path sessionDir = getSessionDir();

    // Настройки поиска
    private String defaultSearchText = "Java Developer";
    private String areaCode = "113"; // Россия

    // Настройки браузера
    private boolean browserHeadless = true;
    private int pageTimeout = 30000;

    private static Path getSessionDir() {
        // Сохраняем сессию в корне проекта
        String projectRoot = System.getProperty("user.dir");
        return Paths.get(projectRoot);
    }

    /**
     * Путь к файлу сессии Playwright.
     */
    public Path getSessionFile() {
        return sessionDir.resolve("hh_session.json");
    }

    /**
     * Проверка наличия файла сессии.
     */
    public boolean hasSessionFile() {
        return Files.exists(getSessionFile());
    }
}
