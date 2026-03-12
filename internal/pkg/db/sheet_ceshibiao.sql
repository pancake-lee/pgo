DROP TABLE IF EXISTS `sheet_ceshibiao`;

CREATE TABLE IF NOT EXISTS `sheet_ceshibiao` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `last_edit_from` varchar(200) NOT NULL DEFAULT "" COMMENT "ltbl/mtbl, 避免循环触发同步逻辑",

  `jingtou` varchar(255) NOT NULL DEFAULT "" COMMENT 'source[镜头/SingleText]',
  `changci` varchar(255) NOT NULL DEFAULT "" COMMENT 'source[场次/SingleText]',
  `zhuangtai` varchar(191) NOT NULL DEFAULT "" COMMENT 'source[状态/SingleSelect]',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='192.168.17.29/spcoWylchuWu3/dstaBbsMatBLqc84Bh';

-- source datasheetId: dstaBbsMatBLqc84Bh
