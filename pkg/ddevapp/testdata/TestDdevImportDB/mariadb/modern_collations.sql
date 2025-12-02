-- MySQL dump with modern collations that should be replaced
-- MariaDB 11.x uses utf8mb4_uca1400_ai_ci
-- MySQL 8.0+ uses utf8mb4_0900_ai_ci
-- These should be replaced with utf8mb4_unicode_ci

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;

--
-- Table with MariaDB 11.x modern collation
--

DROP TABLE IF EXISTS `modern_mariadb`;
CREATE TABLE `modern_mariadb` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `title` varchar(255) COLLATE utf8mb4_uca1400_ai_ci NOT NULL,
  `description` text COLLATE utf8mb4_uca1400_ai_ci,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;

INSERT INTO `modern_mariadb` VALUES (1,'Test Title','Test Description');

--
-- Table with MySQL 8.0+ modern collation
--

DROP TABLE IF EXISTS `modern_mysql`;
CREATE TABLE `modern_mysql` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(100) COLLATE utf8mb4_0900_ai_ci NOT NULL,
  `email` varchar(255) COLLATE utf8mb4_0900_ai_ci,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO `modern_mysql` VALUES (1,'John Doe','john@example.com');

/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;