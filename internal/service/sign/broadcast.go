package sign

/*func (s *Service) broadcast(request proto.Message, requestType types.RequestType) error {
	details, err := cosmostypes.NewAnyWithValue(request)
	if err != nil {
		return err
	}

	for _, party := range s.tssP.Parties {
		_, err = s.con.SignAndSubmit(context.TODO(), party.Address, &types.MsgSubmitRequest{
			Type:    requestType,
			Details: details,
		})

		if err != nil {
			s.log.WithError(err).Error("error submitting acceptance")
		}
	}

	return nil
}
*/
