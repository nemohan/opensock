use im;
grant all privileges on im to 'lumosim'@'localhost' identified by 'Lumosim2017';
create table if not exists user(
	uid bigint,
	name varchar(256),
	level int,
	icon varchar(1024),
	state varchar(48),
	score int,
	lastlogin bigint,
	primary key(uid)

)engine=InnoDB character set utf8;

create table if not exists rostergroups(
	pair varchar(128),
	uid bigint,
	grp varchar(128),
)engine=InnoDB character set utf8;


create table if not exists offline_msg(
	fromid bigint,
	toid bigint,
	msg varchar(1024),
	time varchar(128)
)engine=InnoDB character set utf8;

create table if not exists apply(
	fromid bigint,
	toid bigint,
	msg varchar(1024)

)engine=InnoDB character set utf8;
