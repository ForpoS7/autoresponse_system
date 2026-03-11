package by.icemens.hh_aggregate_service.controller;

import by.icemens.hh_aggregate_service.dto.VacancyResponse;
import by.icemens.hh_aggregate_service.entity.Vacancy;
import by.icemens.hh_aggregate_service.service.ParserService;
import by.icemens.hh_aggregate_service.service.TokenService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.web.bind.annotation.*;

import java.util.List;
import java.util.stream.Collectors;

@RestController
@RequestMapping("/api")
@RequiredArgsConstructor
public class ParserController {

    private final ParserService parserService;
    private final TokenService tokenService;

    @GetMapping("/vacancies")
    public ResponseEntity<List<VacancyResponse>> getVacancies(
            @RequestParam(defaultValue = "Java Developer") String query,
            @RequestParam(defaultValue = "0") int page,
            @AuthenticationPrincipal UserDetails userDetails
    ) {
        Long userId = parserService.getCurrentUserId(userDetails);
        List<Vacancy> vacancies = parserService.parseVacancies(query, page, userId);

        List<VacancyResponse> response = vacancies.stream()
                .map(parserService::toResponse)
                .collect(Collectors.toList());

        return ResponseEntity.ok(response);
    }

    /**
     * Извлечение hhtoken из cookies браузера
     * Открывает hh.ru и автоматически извлекает токен авторизации
     */
    @PostMapping("/hh-token")
    public ResponseEntity<Void> extractAndSaveToken(
            @AuthenticationPrincipal UserDetails userDetails
    ) {
        Long userId = parserService.getCurrentUserId(userDetails);
        parserService.extractAndSaveToken(userId);
        return ResponseEntity.ok().build();
    }
}
