package by.icemens.hh_aggregate_service;

import by.icemens.hh_aggregate_service.config.Settings;
import by.icemens.hh_aggregate_service.model.Vacancy;
import by.icemens.hh_aggregate_service.service.BrowserManager;
import by.icemens.hh_aggregate_service.service.VacancySearchService;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.List;

/**
 * Демонстрация парсинга вакансий HH.ru.
 * Перед запуском выполните Login.main() для авторизации.
 */
public class VacancyParserDemo {

    private static final Logger logger = LoggerFactory.getLogger(VacancyParserDemo.class);

    public static void main(String[] args) {
        Settings settings = new Settings();

        // Проверка сессии
        if (!settings.hasSessionFile()) {
            logger.error("Сессия не найдена!");
            logger.error("Запустите Login.main() для авторизации.");
            logger.error("Путь к сессии: {}", settings.getSessionFile().toAbsolutePath());
            System.exit(1);
        }

        logger.info("============================================================");
        logger.info("           Парсинг вакансий HH.ru");
        logger.info("============================================================");

        String query = args.length > 0 ? args[0] : settings.getDefaultSearchText();
        int page = args.length > 1 ? Integer.parseInt(args[1]) : 0;

        logger.info("Запрос: '{}', страница: {}", query, page);

        try (BrowserManager browserManager = new BrowserManager(settings)) {
            VacancySearchService searchService = new VacancySearchService(settings, browserManager);

            List<Vacancy> vacancies = searchService.search(query, page);

            if (vacancies.isEmpty()) {
                logger.warn("Вакансии не найдены.");
                return;
            }

            logger.info("Найдено вакансий: {}", vacancies.size());

            for (int i = 0; i < vacancies.size(); i++) {
                Vacancy v = vacancies.get(i);
                logger.info("{}. {}", i + 1, v.getTitle());
                logger.info("   Компания: {}", v.getEmployer());
                logger.info("   URL: {}", v.getUrl());
            }

        } catch (Exception e) {
            logger.error("Ошибка: {}", e.getMessage(), e);
            System.exit(1);
        }
    }
}
