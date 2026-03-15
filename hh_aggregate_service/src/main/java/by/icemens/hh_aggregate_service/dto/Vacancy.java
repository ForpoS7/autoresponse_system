package by.icemens.hh_aggregate_service.dto;

import lombok.*;

@Data
@Builder
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
public class Vacancy {

    private Long id;

    private String title;

    private String url;

    private String employer;

    private String description;

    private Integer salaryFrom;

    private Integer salaryTo;

    private String currency;

    private String region;

    private Long userId;
}
