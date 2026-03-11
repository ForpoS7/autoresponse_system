package by.icemens.hh_aggregate_service.message;

import lombok.Builder;
import lombok.Getter;
import lombok.Setter;

import java.time.LocalDateTime;

@Builder
@Setter
@Getter
public class VacancyMessage {

    private Long vacancyId;
    private String title;
    private String employer;
    private String url;
    private LocalDateTime parsedAt;
    private Long userId;

}
