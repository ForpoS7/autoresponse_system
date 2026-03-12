package by.icemens.hh_aggregate_service.controller;

import by.icemens.hh_aggregate_service.dto.Vacancy;
import by.icemens.hh_aggregate_service.service.CustomUserDetailsService;
import by.icemens.hh_aggregate_service.service.ParserService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.web.bind.annotation.*;

import java.util.List;

@RestController
@RequestMapping("/api")
@RequiredArgsConstructor
public class ParserController {

    private final CustomUserDetailsService userDetailsService;
    private final ParserService parserService;

    /**
     * Парсинг вакансий
     * Открывает hh.ru и автоматически парсит вакансии
     */
    @GetMapping("/vacancies")
    public ResponseEntity<List<Vacancy>> getVacancies(
            @RequestParam(defaultValue = "Java Developer") String query,
            @RequestParam(defaultValue = "0") int page,
            @AuthenticationPrincipal UserDetails userDetails
    ) {
        Long userId = userDetailsService.getCurrentUserId(userDetails);
        List<Vacancy> response = parserService.parseVacancies(query, page, userId);

        return ResponseEntity.ok(response);
    }
}
