package by.icemens.hh_aggregate_service.dto;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class VacancyResponse {

    private Long id;
    private String title;
    private String url;
    private String employer;
    private String description;
    private Integer salaryFrom;
    private Integer salaryTo;
    private String currency;
    private String region;
}
