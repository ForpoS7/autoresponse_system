package by.icemens.hh_aggregate_service.config;

import lombok.Getter;
import lombok.Setter;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.HashMap;
import java.util.Map;

@Component
@Slf4j
@Getter
@Setter
public class PlaywrightConfig {
    @Value("${hh.area-code:113}")
    private String areaCode;

    @Value("${hh.browser.headless:true}")
    private boolean headless;

    public static final Map<String, String> DEFAULT_HEADERS = new HashMap<>();
    static {
        DEFAULT_HEADERS.put("User-Agent", 
            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36");
        DEFAULT_HEADERS.put("Accept", 
            "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8");
        DEFAULT_HEADERS.put("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7");
        DEFAULT_HEADERS.put("Sec-Ch-Ua", 
            "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"120\", \"Google Chrome\";v=\"120\"");
        DEFAULT_HEADERS.put("Sec-Ch-Ua-Mobile", "?0");
        DEFAULT_HEADERS.put("Sec-Ch-Ua-Platform", "\"Windows\"");
    }
}
