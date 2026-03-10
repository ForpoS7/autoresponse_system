package by.icemens.hh_aggregate_service.model;

import com.fasterxml.jackson.annotation.JsonProperty;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

/**
 * Модель данных вакансии.
 */
@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class Vacancy {
    
    /**
     * Заголовок вакансии.
     */
    private String title;
    
    /**
     * URL вакансии.
     */
    @JsonProperty("url")
    private String url;
    
    /**
     * Работодатель.
     */
    private String employer;
    
    /**
     * Описание вакансии.
     */
    private String description;
    
    /**
     * Зарплатная вилка (минимум).
     */
    @JsonProperty("salary_from")
    private Integer salaryFrom;
    
    /**
     * Зарплатная вилка (максимум).
     */
    @JsonProperty("salary_to")
    private Integer salaryTo;
    
    /**
     * Валюта зарплаты.
     */
    private String currency;
    
    /**
     * Регион.
     */
    private String region;
}
