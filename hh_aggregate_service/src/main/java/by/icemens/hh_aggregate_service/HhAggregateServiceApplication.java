package by.icemens.hh_aggregate_service;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.scheduling.annotation.EnableScheduling;

@SpringBootApplication
@EnableScheduling
public class HhAggregateServiceApplication {
    public static void main(String[] args) {
        SpringApplication.run(HhAggregateServiceApplication.class, args);
    }
}
