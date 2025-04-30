package test

import (
	"fmt"
	"time"

	"github.com/NeilXu2017/landau/log"
	"github.com/google/uuid"

	"github.com/NeilXu2017/landau/data"
)

type (
	zone struct {
		ZoneID      int    `db:"zone_id"`
		AzGroup     int    `db:"az_group"`
		ZoneCode    string `db:"zone_name"`
		ZoneName    string `db:"c_name"`
		AzGroupCode string `db:"az_group_name"`
		Bit         int    `db:"bit"`
		Valid       int    `db:"valid"`
		CreateTime  int    `db:"create_time"`
		UpdateTime  int    `db:"update_time"`
	}
	_EipUlbInfo struct {
		PublicIp string `db:"public_ip"`
		EipId    string `db:"eip_id"`
		UlbId    string `db:"ulb_id"`
		Name     string
		Year     int
	}
	_TConfigInfo struct {
		Idx         int    `db:"idx"`
		ConfigItem  string `db:"config_item"`
		ConfigValue string `db:"config_value"`
		CreateTime  int    `db:"create_time"`
	}
)

// String 格式化输出
func (c *zone) String() string {
	return fmt.Sprintf("zone_id=%d zone_code=%s zone_name=%s region_id=%d region_code=%s bit=%d valid=%d create_time=%d update_time=%d", c.ZoneID, c.ZoneCode, c.ZoneName, c.AzGroup, c.AzGroupCode, c.Bit, c.Valid, c.CreateTime, c.UpdateTime)
}

// InitDB 设置缺省DB连接参数
func InitDB() {
	data.NewDefaultDatabase("192.168.1.10", 3306, "x", "xxx", "xdb")
	data.SetEnableOptimizeConnection(true)
	data.SetDbPingCheckInterval(5)
	data.SetOptimizeConnectionTrace(false)
	data.SetBadConnectionCount(6)
	data.SetOptimizeConnectionEventCallback(func(dbConn, eventType string, t int64) {
		log.Info("[OptimizeConnectionEventCallback] eveType:%s time:%d db:{%s}", eventType, t, dbConn)
	})
}

// CheckDB 测试DB访问
func CheckDB() {
	strSQL := "select zone_id,az_group,ifnull(zone_name,'') as zone_name,ifnull(c_name,'') as c_name,ifnull(az_group_name,'') as az_group_name,bit,valid,create_time,update_time from t_zone where zone_id=?"
	z := zone{}
	_, _ = data.Get(&z, strSQL, 10018)

	var zs []zone
	strSQL = "select zone_id,az_group,ifnull(zone_name,'') as zone_name,ifnull(c_name,'') as c_name,ifnull(az_group_name,'') as az_group_name,bit,valid,create_time,update_time from t_zone"
	_, _ = data.Gets(&zs, strSQL)

	c := data.NewDatabase2("192.168.154.15", 3306, "xxx", "xxx", "xdb")
	strSQL = "insert into t_configs (cfg_key,cfg_value,cfg_memo,create_time,update_time) values (?,?,?,NOW(),NOW())"
	idx, _ := c.Insert(strSQL, "test_db.insert.1", "{hello}", "test")
	_, _ = c.Insert(strSQL, "test_db.insert.2", "{hello world}", "test")
	strSQL = "update t_configs set cfg_value=?,update_time=NOW() where idx=?"
	_, _ = c.Exec(strSQL, "update hello world", idx)
}

// CheckSQLInterpolate 测试MYSQL SQL处理
func CheckSQLInterpolate() {
	strSQL := "select zone_id,az_group,ifnull(zone_name,'') as zone_name,ifnull(c_name,'') as c_name,ifnull(az_group_name,'') as az_group_name,bit,valid,create_time,update_time from t_zone where zone_id=?"
	checkSQL, err := data.GetBuiltSQL(strSQL, 10018)
	log.Info("SQL:%s error:%v", checkSQL, err)
}

// CheckExecTx  测试事务
func CheckExecTx() {
	c := data.NewDatabase2("192.168.154.15", 3306, "xxx", "xxx", "xdb")
	bizFunc := func(c data.Execute) error {
		strSQL := "insert into t_configs (cfg_key,cfg_value,cfg_memo,create_time,update_time) values (?,?,?,NOW(),NOW())"
		_, idx, err := c.Exec(strSQL, "test_db.insert.tx.1", "{hello}", "test")
		if err != nil {
			return err
		}
		_, _, err = c.Exec(strSQL, "test_db.insert.tx.2", "{hello world}", "test")
		if err != nil {
			return err
		}
		strSQL = "update t_configs set cfg_value=?,update_time=NOW() where idx=?"
		_, _, err = c.Exec(strSQL, "update hello world", idx)
		return err
	}
	_ = c.ExecTx(bizFunc)
}

func CheckQueryInExecTx() {
	db := data.NewDatabase2("127.0.0.1", 3306, "xxx", "xxx", "xdb")
	bizFunc := func(c data.Execute) error {
		sId, t, regionId := uuid.NewString(), time.Now().Unix(), "us-ca"
		sql := `UPDATE t_xxx_pool SET status=1,session_id=?,modify_time=? WHERE pay_mode=0 AND status=0 AND region=? AND eip_type=0 ORDER BY modify_time, public_ip LIMIT 1`
		rowAffected, _, err := c.Exec(sql, sId, t, regionId)
		if err != nil {
			return err
		}
		if rowAffected != 1 {
			return fmt.Errorf("[UGABindUPath] no free eip (paymode=0 status=0 region=%s eip_type=0)", regionId)
		}
		sql = `SELECT eip_id,public_ip,lb_id FROM t_xxx_pool WHERE status=1 AND region=? AND session_id=?`
		m := _EipUlbInfo{Name: "neil", Year: 18}
		rowAffected, err = c.Get(&m, sql, regionId, sId)
		if err != nil {
			return err
		}
		if rowAffected != 1 {
			return fmt.Errorf("[UGABindUPath] fetch empty by (%s,%s)", regionId, sId)
		}
		log.Info("[CheckQueryInExecTx] Get :%v", m)
		var n []_EipUlbInfo
		_, _ = c.Gets(&n, `SELECT eip_id,public_ip,ulb_id FROM t_xxx_pool WHERE region=? `, regionId)
		for _, d := range n {
			log.Info("[CheckQueryInExecTx] Gets :%v", d)
		}
		sql = `INSERT INTO t_xxx (id,line_id,eip_id,create_time,delete_time,status) VALUES (?,?,?,?,0,0)`
		rowAffected, _, err = c.Exec(sql, "xxx", "yyy", m.EipId, t)
		if err != nil {
			return err
		}
		if rowAffected != 1 {
			return fmt.Errorf("[UGABindUPath] insert t_ugaa_binding eip_id=%s failure", m.EipId)
		}
		return nil
	}
	_ = db.ExecTx(bizFunc)
}

func CheckDBTimeout() {
	sql := `SELECT idx,config_item,config_value,create_time FROM t_configs WHERE config_item=?`
	m := _TConfigInfo{}
	_, err := data.Get(&m, sql, "cpu.memory.extra.sku")
	if err != nil {
		log.Error("[CheckDBTimeout] Get err:%v", err)
	} else {
		log.Info("[CheckDBTimeout] Config:%+v", m)
	}
}
