package services

import (
	"context"

	"git.solsynth.dev/hydrogen/dealer/pkg/hyper"
	"git.solsynth.dev/hydrogen/dealer/pkg/proto"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/database"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/gap"
	"git.solsynth.dev/hydrogen/messaging/pkg/internal/models"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

func GetRealmWithExtID(id uint) (models.Realm, error) {
	var realm models.Realm
	pc, err := gap.H.GetServiceGrpcConn(hyper.ServiceTypeAuthProvider)
	if err != nil {
		return realm, err
	}
	response, err := proto.NewRealmClient(pc).GetRealm(context.Background(), &proto.LookupRealmRequest{
		Id: lo.ToPtr(uint64(id)),
	})
	if err != nil {
		return realm, err
	}
	prefix := viper.GetString("database.prefix")
	rm, err := hyper.LinkRealm(database.C, prefix+"realms", response)
	return models.Realm{BaseRealm: rm}, err
}

func GetRealmWithAlias(alias string) (models.Realm, error) {
	var realm models.Realm
	pc, err := gap.H.GetServiceGrpcConn(hyper.ServiceTypeAuthProvider)
	if err != nil {
		return realm, err
	}
	response, err := proto.NewRealmClient(pc).GetRealm(context.Background(), &proto.LookupRealmRequest{
		Alias: &alias,
	})
	if err != nil {
		return realm, err
	}
	prefix := viper.GetString("database.prefix")
	rm, err := hyper.LinkRealm(database.C, prefix+"realms", response)
	return models.Realm{BaseRealm: rm}, err
}

func GetRealmMember(realmId uint, userId uint) (*proto.RealmMemberInfo, error) {
	pc, err := gap.H.GetServiceGrpcConn(hyper.ServiceTypeAuthProvider)
	if err != nil {
		return nil, err
	}
	response, err := proto.NewRealmClient(pc).GetRealmMember(context.Background(), &proto.RealmMemberLookupRequest{
		RealmId: lo.ToPtr(uint64(realmId)),
		UserId:  lo.ToPtr(uint64(userId)),
	})
	if err != nil {
		return nil, err
	} else {
		return response, nil
	}
}

func ListRealmMember(realmId uint) ([]*proto.RealmMemberInfo, error) {
	pc, err := gap.H.GetServiceGrpcConn(hyper.ServiceTypeAuthProvider)
	if err != nil {
		return nil, err
	}
	response, err := proto.NewRealmClient(pc).ListRealmMember(context.Background(), &proto.RealmMemberLookupRequest{
		RealmId: lo.ToPtr(uint64(realmId)),
	})
	if err != nil {
		return nil, err
	} else {
		return response.Data, nil
	}
}
