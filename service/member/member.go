package member

import (
	"context"

	"github.com/paper-trade-chatbot/be-proto/member"
	"github.com/paper-trade-chatbot/be-order/config"
)

type MemberIntf interface {
	CreateMember(ctx context.Context, in *member.CreateMemberReq) (*member.CreateMemberRes, error)
	GetMember(ctx context.Context, in *member.GetMemberReq) (*member.GetMemberRes, error)
	GetMembers(ctx context.Context, in *member.GetMembersReq) (*member.GetMembersRes, error)
	ModifyMember(ctx context.Context, in *member.ModifyMemberReq) (*member.ModifyMemberRes, error)
	ResetPassword(ctx context.Context, in *member.ResetPasswordReq) (*member.ResetPasswordRes, error)
	DeleteMember(ctx context.Context, in *member.DeleteMemberReq) (*member.DeleteMemberRes, error)
}

type MemberImpl struct {
	MemberClient member.MemberServiceClient
}

var (
	MemberServiceHost    = config.GetString("MEMBER_GRPC_HOST")
	MemberServerGRpcPort = config.GetString("MEMBER_GRPC_PORT")
)

func New(memberClient member.MemberServiceClient) MemberIntf {
	return &MemberImpl{
		MemberClient: memberClient,
	}
}

func (impl *MemberImpl) CreateMember(ctx context.Context, in *member.CreateMemberReq) (*member.CreateMemberRes, error) {
	return impl.MemberClient.CreateMember(ctx, in)
}

func (impl *MemberImpl) GetMember(ctx context.Context, in *member.GetMemberReq) (*member.GetMemberRes, error) {
	return impl.MemberClient.GetMember(ctx, in)
}

func (impl *MemberImpl) GetMembers(ctx context.Context, in *member.GetMembersReq) (*member.GetMembersRes, error) {
	return impl.MemberClient.GetMembers(ctx, in)
}

func (impl *MemberImpl) ModifyMember(ctx context.Context, in *member.ModifyMemberReq) (*member.ModifyMemberRes, error) {
	return impl.MemberClient.ModifyMember(ctx, in)
}

func (impl *MemberImpl) ResetPassword(ctx context.Context, in *member.ResetPasswordReq) (*member.ResetPasswordRes, error) {
	return impl.MemberClient.ResetPassword(ctx, in)
}

func (impl *MemberImpl) DeleteMember(ctx context.Context, in *member.DeleteMemberReq) (*member.DeleteMemberRes, error) {
	return impl.MemberClient.DeleteMember(ctx, in)
}
