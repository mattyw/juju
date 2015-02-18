// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state

import (
	"github.com/juju/errors"
	"github.com/juju/loggo"
	jujutxn "github.com/juju/txn"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/mgo.v2/txn"
)

var meterStatusLogger = loggo.GetLogger("juju.state.meterstatus")

// MeterStatusCode represents the meter status code of a unit.
type MeterStatusCode string

const (
	MeterNotSet       MeterStatusCode = "NOT SET"
	MeterNotAvailable MeterStatusCode = "NOT AVAILABLE"
	MeterGreen        MeterStatusCode = "GREEN"
	MeterAmber        MeterStatusCode = "AMBER"
	MeterRed          MeterStatusCode = "RED"
)

type meterStatusDoc struct {
	DocID   string          `bson:"_id"`
	EnvUUID string          `bson:"env-uuid"`
	Code    MeterStatusCode `bson:"code"`
	Info    string          `bson:"info"`
}

// SetMeterStatus sets the meter status for the unit.
func (u *Unit) SetMeterStatus(codeRaw, info string) error {
	code := MeterStatusCode(codeRaw)
	switch code {
	case MeterGreen, MeterAmber, MeterRed:
	default:
		return errors.Errorf("invalid meter status %q", code)
	}
	meterDoc, err := u.getMeterStatusDoc()
	if err != nil {
		return errors.Annotatef(err, "cannot update meter status for unit %s", u.Name())
	}
	if meterDoc.Code == code && meterDoc.Info == info {
		return nil
	}

	buildTxn := func(attempt int) ([]txn.Op, error) {
		if attempt > 0 {
			err := u.Refresh()
			if err != nil {
				return nil, errors.Trace(err)
			}
			meterDoc, err = u.getMeterStatusDoc()
			if err != nil {
				return nil, errors.Annotatef(err, "cannot update meter status for unit %s", u.Name())
			}
			if meterDoc.Code == code && meterDoc.Info == info {
				return nil, jujutxn.ErrNoOperations
			}
		}
		return setMeterStatusOp(u, u.st, u.globalKey(), code, info), nil
	}
	return errors.Annotatef(u.st.run(buildTxn), "cannot set meter state for unit %s", u.Name())
}

// TODO (mattyw) Remove the call from the metricmanager - make it use this call.
// TODO (mattyw) Do this in a single transaction - only on commercial charms
func (st *State) SetMeterStatusOnAllUnits(codeRaw, info string) error {
	var ops []txn.Op
	services, err := st.AllServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		units, err := svc.AllUnits()
		if err != nil {
			return err
		}
		for _, unit := range units {
			ops = append(ops, setMeterStatusOp(unit, st, unit.globalKey(), MeterStatusCode(codeRaw), info)...)
		}
	}
	return errors.Annotatef(st.runTransaction(ops), "cannot set meter state for all units")
}

// setMeterStatusOp returns the operation needed to set the meter status
// document associated with the given globalKey.
func setMeterStatusOp(u *Unit, st *State, globalKey string, code MeterStatusCode, info string) []txn.Op {
	return []txn.Op{
		{
			C:      unitsC,
			Id:     u.doc.DocID,
			Assert: isAliveDoc,
		}, {
			C:      meterStatusC,
			Id:     st.docID(globalKey),
			Assert: txn.DocExists,
			Update: bson.D{{"$set", bson.D{{"code", code}, {"info", info}}}},
		}}
}

// createMeterStatusOp returns the operation needed to create the meter status
// document associated with the given globalKey.
func createMeterStatusOp(st *State, globalKey string, doc *meterStatusDoc) txn.Op {
	doc.EnvUUID = st.EnvironUUID()
	return txn.Op{
		C:      meterStatusC,
		Id:     st.docID(globalKey),
		Assert: txn.DocMissing,
		Insert: doc,
	}
}

// removeMeterStatusOp returns the operation needed to remove the meter status
// document associated with the given globalKey.
func removeMeterStatusOp(st *State, globalKey string) txn.Op {
	return txn.Op{
		C:      meterStatusC,
		Id:     st.docID(globalKey),
		Remove: true,
	}
}

// GetMeterStatus returns the meter status for the unit.
func (u *Unit) GetMeterStatus() (code, info string, err error) {
	status, err := u.getMeterStatusDoc()
	if err != nil {
		return string(MeterNotAvailable), "", errors.Annotatef(err, "cannot retrieve meter status for unit %s", u.Name())
	}
	return string(status.Code), status.Info, nil
}

func (u *Unit) getMeterStatusDoc() (*meterStatusDoc, error) {
	meterStatuses, closer := u.st.getCollection(meterStatusC)
	defer closer()
	var status meterStatusDoc
	err := meterStatuses.FindId(u.globalKey()).One(&status)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &status, nil
}
