//package by.icemens.hh_aggregate_service.service;
//
//import by.icemens.hh_aggregate_service.entity.Vacancy;
//import lombok.RequiredArgsConstructor;
//import lombok.extern.slf4j.Slf4j;
//import org.springframework.scheduling.annotation.Scheduled;
//import org.springframework.stereotype.Service;
//
//import java.util.List;
//
//@Service
//@RequiredArgsConstructor
//@Slf4j
//public class SchedulerService {
//
////    private final HhParserService hhParserService;
//
//    // Отключено по умолчанию, включается через application.yml
////    @Scheduled(cron = "${scheduler.parser.cron:-}")
//    public void scheduledParser() {
//        log.info("Запуск планового парсинга вакансий");
//
//        // Для MVP используем userId=1 и запрос по умолчанию
//        // В реальном приложении нужно хранить настройки парсинга для каждого пользователя
//        try {
//            List<Vacancy> vacancies = hhParserService.parseVacancies("Java Developer", 0, 1L);
//            log.info("Плановый парсинг завершён. Найдено вакансий: {}", vacancies.size());
//        } catch (Exception e) {
//            log.error("Ошибка при плановом парсинге: {}", e.getMessage(), e);
//        }
//    }
//}
