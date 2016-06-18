SET NAMES utf8;
SET time_zone = '+00:00';
SET foreign_key_checks = 0;
SET sql_mode = 'NO_AUTO_VALUE_ON_ZERO';

DROP TABLE IF EXISTS `catalogo`;
CREATE TABLE `catalogo` (
      `id` int(11) NOT NULL AUTO_INCREMENT,
      `nombre` varchar(80) NOT NULL,
      `valor` varchar(80) NOT NULL,
      `tipo` varchar(40) NOT NULL,
      `descripcion` tinytext NOT NULL,
      PRIMARY KEY (`id`)

) ENGINE=InnoDB DEFAULT CHARSET=utf8;


DROP TABLE IF EXISTS `padron`;
CREATE TABLE `padron` (
      `nombre` varchar(80) NOT NULL,
      `telegram_uid` varchar(60) NOT NULL,
      `telegram_gid` varchar(60) NOT NULL,
      `alias_secreto` varchar(60) NOT NULL,
      UNIQUE KEY `telegram_uid` (`telegram_uid`)

) ENGINE=InnoDB DEFAULT CHARSET=utf8;


DROP TABLE IF EXISTS `planilla`;
CREATE TABLE `planilla` (
      `id` int(11) NOT NULL AUTO_INCREMENT,
      `id_votacion` int(11) NOT NULL,
      `titular` varchar(80) NOT NULL,
      `idoficial` varchar(40) NOT NULL DEFAULT '' COMMENT 'Alguna identidicación en caso de ser necesario distinguir al titular',
      `descripcion` tinytext NOT NULL,
      PRIMARY KEY (`id`),
      KEY `id_votacion` (`id_votacion`),
      CONSTRAINT `planilla_ibfk_1` FOREIGN KEY (`id_votacion`) REFERENCES `votacion` (`id`) ON DELETE CASCADE

) ENGINE=InnoDB DEFAULT CHARSET=utf8;


DROP TABLE IF EXISTS `votacion`;
CREATE TABLE `votacion` (
      `id` int(11) NOT NULL AUTO_INCREMENT,
      `titulo` varchar(255) NOT NULL,
      `descripcion` tinytext NOT NULL,
      `inicio` datetime NOT NULL,
      `fin` datetime NOT NULL,
      `secreto` varchar(255) NOT NULL,
      `estatus` tinyint(4) NOT NULL DEFAULT '1' COMMENT '1- agendada, 2-abierta, 3-cerrada, 4-cancelada',
      PRIMARY KEY (`id`)

) ENGINE=InnoDB DEFAULT CHARSET=utf8;


DROP TABLE IF EXISTS `votacion_catalogo`;
CREATE TABLE `votacion_catalogo` (
      `id_votacion` int(11) NOT NULL,
      `id_catalogo` int(11) NOT NULL,
      `significado` varchar(255) NOT NULL COMMENT 'Significado de este valor en el contexto de la votación al que se destina',
      PRIMARY KEY (`id_votacion`,`id_catalogo`),
      KEY `id_catalogo` (`id_catalogo`),
      CONSTRAINT `votacion_catalogo_ibfk_1` FOREIGN KEY (`id_votacion`) REFERENCES `votacion` (`id`),
      CONSTRAINT `votacion_catalogo_ibfk_2` FOREIGN KEY (`id_catalogo`) REFERENCES `catalogo` (`id`)

) ENGINE=InnoDB DEFAULT CHARSET=utf8;


DROP TABLE IF EXISTS `voto`;
CREATE TABLE `voto` (
      `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
      `id_votacion` int(11) NOT NULL,
      `id_planilla` int(11) NOT NULL,
      `valor` varchar(80) NOT NULL,
      `alias_secreto` varchar(255) NOT NULL,
      `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (`id`),
      UNIQUE KEY `id_votacion_alias_secreto` (`id_votacion`,`alias_secreto`),
      KEY `id_planilla` (`id_planilla`),
      KEY `id_catalogo` (`valor`),
      KEY `alias_secreto` (`alias_secreto`),
      CONSTRAINT `voto_ibfk_1` FOREIGN KEY (`id_votacion`) REFERENCES `votacion` (`id`),
      CONSTRAINT `voto_ibfk_2` FOREIGN KEY (`id_planilla`) REFERENCES `planilla` (`id`)

) ENGINE=InnoDB DEFAULT CHARSET=utf8;


