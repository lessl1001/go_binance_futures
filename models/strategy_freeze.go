package models

import (
    "github.com/beego/beego/v2/client/orm"
)

func DeleteStrategyFreeze(id int64) error {
    o := orm.NewOrm()
    _, err := o.Raw("DELETE FROM strategy_freeze WHERE id = ?", id).Exec()
    return err
}
