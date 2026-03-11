package by.icemens.hh_aggregate_service.repository;

import by.icemens.hh_aggregate_service.entity.HhToken;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

import java.util.Optional;

@Repository
public interface HhTokenRepository extends JpaRepository<HhToken, Long> {

    Optional<HhToken> findByUserId(Long userId);
}
