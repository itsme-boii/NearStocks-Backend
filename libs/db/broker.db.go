package db

type BrokerDB struct{}

func (*BrokerDB) Create(name string) *BrokerTable {
	broker := BrokerTable{Name: name}
	db.Create(&broker)
	return &broker
}

// NOTE: Do not expose for non accessible broker id in apis
func (*BrokerDB) DeleteById(id uint) {
	db.Delete(&BrokerTable{}, id)
}

// NOTE: Do not expose for non accessible broker id in apis
func (*BrokerDB) GetById(id uint) *BrokerTable {
	broker := BrokerTable{}
	return GetDbObjOrNil(db.First(&broker, id), &broker)
}

// NOTE: Do not expose this info via apis to brokers / users
func (*BrokerDB) GetAll() *[]BrokerTable {
	brokers := []BrokerTable{}
	return GetDBObjOrEmptyList(db.Find(&brokers), &brokers)
}

func (brokerDb *BrokerDB) Update(id uint, name string) *BrokerTable {
	broker := BrokerTable{BaseTable: BaseTable{ID: id}, Name: name}
	return GetDbObjOrNil(db.Updates(&broker), &broker)
}
