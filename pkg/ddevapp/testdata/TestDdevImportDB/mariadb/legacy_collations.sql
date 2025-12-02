-- MySQL dump with legacy collations that should NOT be replaced
-- These are compatible across versions and essential for table structure

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;

--
-- Table with legacy collations that must be preserved
--

DROP TABLE IF EXISTS `legacy_collations`;
CREATE TABLE `legacy_collations` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `token` varchar(64) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL,
  `title` varchar(255) COLLATE utf8mb4_general_ci NOT NULL,
  `description` text COLLATE utf8mb4_unicode_ci,
  `latin1_field` varchar(100) CHARACTER SET latin1 COLLATE latin1_swedish_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `token_idx` (`token`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci COMMENT='Table with various legacy collations';

INSERT INTO `legacy_collations` VALUES (1,'abc123','Test Title','Test Description','Latin1 Text');
INSERT INTO `legacy_collations` VALUES (2,'def456','Another Title','Another Description','More Text');

/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;