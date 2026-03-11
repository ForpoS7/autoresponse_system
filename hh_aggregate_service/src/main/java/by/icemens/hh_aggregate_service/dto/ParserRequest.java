package by.icemens.hh_aggregate_service.dto;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ParserRequest {

    @Builder.Default
    private String query = "Java Developer";

    @Builder.Default
    private Integer page = 0;
}
