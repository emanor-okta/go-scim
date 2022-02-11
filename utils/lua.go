package utils

//
//	Shared Lua Scripts
//
/*
	Useed by redis.getByRange(startIndex int, count int, key, embedded_key string) (interface{}, error)
	Gets either users or groups by range

	get ids - local ids=redis.call('LRange',KEYS[1],ARGV[1],ARGV[2]);
	define outer map, loop through ids - local outer={};local oi=1;for k,id in ipairs(ids) do
	define inner map, get doc for id - local inner={};inner[1]=redis.call('Get',id);
	get embedded values - local i=2;local l=redis.call('HVals',id..KEYS[2]);
	add each value to structure - for k2,v in ipairs(l) do inner[i]=v;i=i+1;end;
	add doc to outer map - outer[oi]=inner;oi=oi+1;end;
	return outer - return outer;
*/
const LUA_GET_BY_RANGE = `local ids=redis.call('LRange',KEYS[1],ARGV[1],ARGV[2]);local outer={};local oi=1;for k,id in ipairs(ids) do local inner={};inner[1]=redis.call('Get',id);local i=2;local l=redis.call('HVals',id..KEYS[2]);for k2,v in ipairs(l) do inner[i]=v;i=i+1;end;outer[oi]=inner;oi=oi+1;end;return outer;`

/*
	Used by redis.getByFilter(name, key, embedded_key string) (interface{}, error)
	Gets a user or a group by name

	get records uuid - local uuid=redis.call('HGet', KEYS[1],ARGV[1]);
	if not found return nil - if not uuid then return nil; end;
	add doc as 1st element - local t={};t[1]=redis.call('Get',uuid);
	get docs embedded elements - local i=2;local l=redis.call('HVals',uuid..KEYS[2]);
	add each element to structure - for k,v in ipairs(l) do t[i]=v;i=i+1;end;return t;
*/
const LUA_GET_BY_FILTER = `local uuid=redis.call('HGet', KEYS[1],ARGV[1]);if not uuid then return nil; end;local t={};t[1]=redis.call('Get',uuid);local i=2;local l=redis.call('HVals',uuid..KEYS[2]);for k,v in ipairs(l) do t[i]=v;i=i+1;end;return t;`

/*
	Used by redis.getByUUID(uuid, embedded_key string) (interface{}, error)
	Get a user or group by UUID

	get doc - local doc=redis.call('Get', KEYS[1]);
	if not found return nil - if not doc then return nil; end;
	add doc as 1st element - local t={};t[1]=doc;
	get embedded elements - local i=2;local l=redis.call('HVals',KEYS[1]..KEYS[2]);
	add each element to structure - for k,v in ipairs(l) do t[i]=v;i=i+1;end;return t;
*/
const LUA_GET_BY_UUID = `local doc=redis.call('Get', KEYS[1]);if not doc then return nil; end;local t={};t[1]=doc;local i=2;local l=redis.call('HVals',KEYS[1]..KEYS[2]);for k,v in ipairs(l) do t[i]=v;i=i+1;end;return t;`

//
//	User Lua Scripts
//
/*
	Used by redis.AddUser(doc []byte, userName, uuid string) error
	Adds a new user

	check if user exists - if redis.call('HExists',KEYS[2],ARGV[2]) == 1 then return nil;end;
	set user doc - redis.call('set', KEYS[1], ARGV[1]);
	set users lookup - redis.call('HSet', KEYS[2], ARGV[2], ARGV[3]);
	set users reverse lookup - redis.call('HSet', KEYS[4], ARGV[3], ARGV[2]);
	add to users list - redis.call('LPush', KEYS[3], ARGV[3]);
	return success - return 1;
*/
const LUA_ADD_USER = `if redis.call('HExists',KEYS[2],ARGV[2]) == 1 then return nil;end;redis.call('set', KEYS[1], ARGV[1]);redis.call('HSet', KEYS[2], ARGV[2], ARGV[3]);redis.call('HSet', KEYS[4], ARGV[3], ARGV[2]);redis.call('LPush', KEYS[3], ARGV[3]);return 1;`

/*
	Used by Redis.UpdateUser((uuid string, doc []byte, active bool, userElement string, ids, groups []string) error
	Update ACTIVE user, add groups

	replace existing user doc - redis.call('Set',KEYS[1],ARGV[1]);
	loop through groups - for i = 4,2*ARGV[3]+2,2 do
	  add group to users groups - redis.call('HSet',KEYS[1].."_groups",ARGV[i],ARGV[i+1]);
	  add user to groups members - redis.call('HSet',ARGV[i].."_members",KEYS[1],ARGV[2]);
	end loop through groups - end;
	return success - return 1;
*/
const LUA_UPDATE_USER_ACTIVE = `redis.call('Set',KEYS[1],ARGV[1]);for i = 4,2*ARGV[3]+2,2 do redis.call('HSet',KEYS[1].."_groups",ARGV[i],ARGV[i+1]);redis.call('HSet',ARGV[i].."_members",KEYS[1],ARGV[2]);end;return 1;`

/*
	Used by Redis.UpdateUser(uuid string, doc []byte, active bool, ids, groups []string) error
	Update IN-ACTIVE user, remove group

	replace existing user doc - redis.call('Set',KEYS[1],ARGV[1]);
	get users groups - local grps=redis.call('HKeys',KEYS[1].."_groups");
	delete users groups map - redis.call('Del',KEYS[1].."_groups");
	remove user from each group - for k,v in ipairs(grps) do redis.call('HDel',v.."_members",KEYS[1]);end;
	return success - return 1;
*/
const LUA_UPDATE_USER_INACTIVE = `redis.call('Set',KEYS[1],ARGV[1]);local grps=redis.call('HKeys',KEYS[1].."_groups");redis.call('Del',KEYS[1].."_groups");for k,v in ipairs(grps) do redis.call('HDel',v.."_members",KEYS[1]);end;return 1;`

/*
	Used by redis.DelUser(uuid string) error
	Deletes a user

	get user name - local n=redis.call('HGet',KEYS[4],ARGV[1]);
	user not found return - if not n then return nil;end;
	delete user lookup - redis.call('HDel',KEYS[2],n);
	delete user reverse lookup - redis.call('HDel',KEYS[4],ARGV[1]);
	delete user doc - redis.call('Del',KEYS[1]);
	remove from user list - redis.call('LRem',KEYS[3],0,ARGV[1]);
	get users groups - local grps=redis.call('HKeys',KEYS[1].."_groups");
	delete users groups map - redis.call('Del',KEYS[1].."_groups");
	remove user from each group - for k,v in ipairs(grps) do redis.call('HDel',v.."_members",ARGV[1]);end;
	return success - return 1;
*/
const LUA_DELETE_USER = `local n=redis.call('HGet',KEYS[4],ARGV[1]);if not n then return nil;end;redis.call('HDel',KEYS[2],n);redis.call('HDel',KEYS[4],ARGV[1]);redis.call('Del',KEYS[1]);redis.call('LRem',KEYS[3],0,ARGV[1]);local grps=redis.call('HKeys',KEYS[1].."_groups");redis.call('Del',KEYS[1].."_groups");for k,v in ipairs(grps) do redis.call('HDel',v.."_members",ARGV[1]);end;return 1;`

/*
	Used by redis.PatchUser(uuid string, userPatch UserPatch) error
	Changes user 'password' or 'active' status

	get user doc - local u=redis.call('Get',KEYS[1]);
	user not found return - if not u then return nil;end;
	change active if included - if KEYS[2]=="true" then u=string.gsub(u,"\"active\":.-,","\"active\":"..ARGV[1]..",");
	  if active set to false remove groups - if ARGV[1]=="false" then
	   get users groups - local grps=redis.call('HKeys',KEYS[1].."_groups");
	   delete users groups map - redis.call('Del',KEYS[1].."_groups");
	   remove user from each group - for k,v in ipairs(grps) do redis.call('HDel',v.."_members",KEYS[1]);end;
	  close if statement for active==false - end;
	close if statement for change active - end;
	change password if included - if KEYS[3]=="true" then u=string.gsub(u,"\"password\":\".-\"","\"password\":\""..ARGV[2].."\"");end;
	save user doc - redis.call('Set',KEYS[1],u);
	return success - return 1;
*/
const LUA_PATCH_USER = `local u=redis.call('Get',KEYS[1]);if not u then return nil;end;if KEYS[2]=="true" then u=string.gsub(u,"\"active\":.-,","\"active\":"..ARGV[1]..",");if ARGV[1]=="false" then local grps=redis.call('HKeys',KEYS[1].."_groups");redis.call('Del',KEYS[1].."_groups");for k,v in ipairs(grps) do redis.call('HDel',v.."_members",KEYS[1]);end;end;end;if KEYS[3]=="true" then u=string.gsub(u,"\"password\":\".-\"","\"password\":\""..ARGV[2].."\"");end;redis.call('Set',KEYS[1],u); return 1; `

//
//	Group Lua Scripts
//
/*
	Used by redis.DelGroup(uuid string) error
	Deletes a group

	get group name - local n = redis.call('HGet',KEYS[1],ARGV[1]);
	get group members - local m = redis.call('HKeys',KEYS[2]);
	foreach members remove this group - for k,v in ipairs(m) do redis.call('HDel',v.."_groups",ARGV[1]);end;
	delete group doc - redis.call('del',KEYS[3]);
	delete from groups lookup - redis.call('HDel',KEYS[4],n);
	remove from groups list - redis.call('LRem',KEYS[5],0,ARGV[1]);
	delete group memebers hash - redis.call('Del',KEYS[2]);
	delete from groups reverse lookup - redis.call('HDel',KEYS[1],ARGV[1]);return 1;
*/
const LUA_DELETE_GROUP = `local n = redis.call('HGet',KEYS[1],ARGV[1]); local m = redis.call('HKeys',KEYS[2]); for k,v in ipairs(m) do redis.call('HDel',v.."_groups",ARGV[1]); end; redis.call('del',KEYS[3]);redis.call('HDel',KEYS[4],n);redis.call('LRem',KEYS[5],0,ARGV[1]);redis.call('Del',KEYS[2]);redis.call('HDel',KEYS[1],ARGV[1]);return 1;`

/*
	Used by redis.UpdateGroupName(uuid string, name string) error
	Updates a group name

	get group name - local k=redis.call('HGet',KEYS[1],ARGV[1]);
	delete group lookup - redis.call('HDel',KEYS[2],k);
	set reverse lookup to name - redis.call('HSet',KEYS[1],ARGV[1],ARGV[2]);
	set group lookup - redis.call('HSet',KEYS[2],ARGV[2],ARGV[1]);
	//define functions to encode/decode - local function code (s) return (string.gsub(s, "\\(.)", function (x) return string.format("\\%03d", string.byte(x)) end)); end local function decode (s) return (string.gsub(s, "\\(%d%d%d)", function (d) return "\\" .. string.char(d) end)); end
	get group doc - local j=redis.call("Get",KEYS[3]);
	//remove " - j=code(j);
	update group name - j = string.gsub(j,"\"displayName\":\".-\"","\"displayName\":\"` + name + `\"");
	//add " - j = decode(j);
	save group doc - redis.call('Set',KEYS[3],j);
	define group snippet with new name - local g='{"value":"'..ARGV[1]..'","display":"'..ARGV[2]..'"}';
	assign snippet to each memembers *_groups hash - local m=redis.call('HKeys',KEYS[4]);for k,v in ipairs(m) do redis.call('HSet',v.."_groups",ARGV[1],g);end; return 1;
*/
const LUA_UPDATE_GROUP_NAME = `local k=redis.call('HGet',KEYS[1],ARGV[1]);redis.call('HDel',KEYS[2],k);redis.call('HSet',KEYS[1],ARGV[1],ARGV[2]);redis.call('HSet',KEYS[2],ARGV[2],ARGV[1]);  local j=redis.call("Get",KEYS[3]); j = string.gsub(j,"\"displayName\":\".-\"","\"displayName\":\""..ARGV[2].."\"");   redis.call('Set',KEYS[3],j); local g='{"value":"'..ARGV[1]..'","display":"'..ARGV[2]..'"}'; local m=redis.call('HKeys',KEYS[4]);for k,v in ipairs(m) do redis.call('HSet',v.."_groups",ARGV[1],g);end; return 1;`
