package by.icemens.hh_aggregate_service.controller;

import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.HashMap;
import java.util.Map;

@RestController
@RequestMapping("/api/scheduler")
@RequiredArgsConstructor
public class SchedulerController {

    @Value("${scheduler.parser.cron:-}")
    private String parserCron;

    @GetMapping("/config")
    public ResponseEntity<Map<String, String>> getSchedulerConfig() {
        Map<String, String> config = new HashMap<>();
        config.put("parserCron", parserCron);
        return ResponseEntity.ok(config);
    }
}
