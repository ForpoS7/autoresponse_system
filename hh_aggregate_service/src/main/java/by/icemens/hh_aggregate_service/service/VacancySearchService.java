package by.icemens.hh_aggregate_service.service;

import by.icemens.hh_aggregate_service.config.Settings;
import by.icemens.hh_aggregate_service.model.Vacancy;
import com.microsoft.playwright.ElementHandle;
import com.microsoft.playwright.Page;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;

/**
 * Сервис для поиска вакансий на HH.ru.
 */
public class VacancySearchService {
    
    private static final Logger logger = LoggerFactory.getLogger(VacancySearchService.class);
    
    private final Settings settings;
    private final BrowserManager browserManager;
    
    public VacancySearchService(Settings settings, BrowserManager browserManager) {
        this.settings = settings;
        this.browserManager = browserManager;
    }
    
    /**
     * Поиск вакансий на HH.ru.
     *
     * @param query текст запроса
     * @param pageNum номер страницы (начиная с 0)
     * @return список вакансий
     */
    public List<Vacancy> search(String query, int pageNum) {
        String searchQuery = query != null ? query : settings.getDefaultSearchText();

        logger.info("Поиск вакансий: запрос='{}', страница={}", searchQuery, pageNum);

        try (BrowserManager.BrowserPage browserPage = browserManager.getPage()) {
            Page page = browserPage.get();

            // Формирование URL для поиска
            String url = String.format(
                "https://hh.ru/search/vacancy?text=%s&area=%s&items_on_page=20&page=%d",
                searchQuery,
                settings.getAreaCode(),
                pageNum
            );

            logger.info("URL: {}", url);
            page.navigate(url);

            // Ожидание результатов
            page.waitForSelector("[data-qa='vacancy-serp__vacancy']");
            
            // Сбор основных данных вакансий
            List<Vacancy> vacancies = new ArrayList<>();
            List<ElementHandle> cards = page.querySelectorAll("[data-qa='vacancy-serp__vacancy']");
            
            for (int i = 0; i < cards.size(); i++) {
                ElementHandle card = cards.get(i);
                try {
                    ElementHandle titleEl = card.querySelector("[data-qa='serp-item__title']");
                    if (titleEl == null) {
                        continue;
                    }
                    
                    String href = titleEl.getAttribute("href");
                    String title = titleEl.textContent().trim();
                    
                    ElementHandle employerEl = card.querySelector(
                        "[data-qa='vacancy-serp__vacancy-employer']"
                    );
                    String employer = employerEl != null 
                        ? employerEl.textContent().trim() 
                        : "Не указан";
                    
                    vacancies.add(Vacancy.builder()
                        .title(title)
                        .url(href)
                        .employer(employer)
                        .build());
                    
                } catch (Exception e) {
                    logger.warn("Не удалось распарсить вакансию {}: {}", i, e.getMessage());
                } finally {
                    if (card != null) {
                        card.dispose();
                    }
                }
            }
            
            logger.info("Найдено вакансий: {}", vacancies.size());
            return vacancies;
            
        } catch (IllegalStateException e) {
            throw e;
        } catch (Exception e) {
            logger.error("Поиск вакансий не удался: {}", e.getMessage(), e);
            throw new RuntimeException("Поиск вакансий не удался: " + e.getMessage(), e);
        }
    }
}
