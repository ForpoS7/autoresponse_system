package by.icemens.hh_aggregate_service.controller;

import by.icemens.hh_aggregate_service.dto.TokenRequest;
import by.icemens.hh_aggregate_service.service.CustomUserDetailsService;
import by.icemens.hh_aggregate_service.service.TokenService;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.security.core.annotation.AuthenticationPrincipal;
import org.springframework.security.core.userdetails.UserDetails;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api")
@RequiredArgsConstructor
public class TokenController {

    private final CustomUserDetailsService userDetailsService;
    private final TokenService tokenService;

    /**
     * Извлечение hhtoken из cookies браузера
     * Открывает hh.ru и автоматически извлекает токен авторизации
     */
    @PostMapping("/hh-token")
    public ResponseEntity<Void> extractAndSaveToken(
            @AuthenticationPrincipal UserDetails userDetails
    ) {
        Long userId = userDetailsService.getCurrentUserId(userDetails);
        tokenService.extractAndSaveToken(userId);
        return ResponseEntity.ok().build();
    }

    /**
     * Получение hhtoken из базы
     */
    @GetMapping("/hh-token")
    public ResponseEntity<TokenRequest> getToken(
            @AuthenticationPrincipal UserDetails userDetails
    ) {
        Long userId = userDetailsService.getCurrentUserId(userDetails);
        var token = tokenService.getToken(userId, userDetails.getUsername());
        return ResponseEntity.ok(token);
    }
}
