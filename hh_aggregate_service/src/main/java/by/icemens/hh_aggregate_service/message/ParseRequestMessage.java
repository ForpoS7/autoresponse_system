package by.icemens.hh_aggregate_service.message;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;

@Data
@Builder
@NoArgsConstructor
@AllArgsConstructor
public class ParseRequestMessage {
    private String jobId;
    private String query;
    private int page;
    private Long userId;
}
