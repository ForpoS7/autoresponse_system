package by.icemens.hh_aggregate_service.controller;

import by.icemens.hh_aggregate_service.dto.HhTokenRequest;
import by.icemens.hh_aggregate_service.dto.VacancyResponse;
import by.icemens.hh_aggregate_service.entity.Vacancy;
//import by.icemens.hh_aggregate_service.service.HhParserService;
import by.icemens.hh_aggregate_service.repository.UserRepository;
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

    private final UserRepository userRepository;
    private final ParserService parserService;
    private final TokenService tokenService;

    @GetMapping("/vacancies")
    public ResponseEntity<List<VacancyResponse>> getVacancies(
            @RequestParam(defaultValue = "Java Developer") String query,
            @RequestParam(defaultValue = "0") int page,
            @AuthenticationPrincipal UserDetails userDetails
    ) {
        Long userId = getCurrentUserId(userDetails);
        List<Vacancy> vacancies = parserService.parseVacancies(query, page, userId);

        List<VacancyResponse> response = vacancies.stream()
                .map(this::toResponse)
                .collect(Collectors.toList());

        return ResponseEntity.ok(response);
    }

    @PostMapping("/hh-token")
    public ResponseEntity<Void> saveHhToken(
            @RequestBody HhTokenRequest request,
            @AuthenticationPrincipal UserDetails userDetails
    ) {
        Long userId = getCurrentUserId(userDetails);
        tokenService.saveHhSession(userId, request.getTokenValue());
        return ResponseEntity.ok().build();
    }

    private Long getCurrentUserId(UserDetails userDetails) {
        // В реальном приложении нужно загружать User из БД по email
        // Для MVP возвращаем заглушку - нужно доработать
        return userRepository.findByEmail(userDetails.getUsername()).orElseThrow(
                () -> new IllegalStateException(
                "Пользователь с таким email - " + userDetails.getUsername() + " не найден.")
        ).getId();
    }

    private VacancyResponse toResponse(Vacancy vacancy) {
        return VacancyResponse.builder()
                .id(vacancy.getId())
                .title(vacancy.getTitle())
                .url(vacancy.getUrl())
                .employer(vacancy.getEmployer())
                .description(vacancy.getDescription())
                .salaryFrom(vacancy.getSalaryFrom())
                .salaryTo(vacancy.getSalaryTo())
                .currency(vacancy.getCurrency())
                .region(vacancy.getRegion())
                .build();
    }
}
