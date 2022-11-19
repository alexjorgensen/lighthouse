

CREATE TABLE `priceData` (
  `priceDate` datetime NOT NULL,
  `sector` varchar(20) NOT NULL DEFAULT '',
  `currency` varchar(20) NOT NULL DEFAULT '',
  `hour` datetime NOT NULL,
  `price` float DEFAULT NULL,
  PRIMARY KEY (`priceDate`,`hour`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8