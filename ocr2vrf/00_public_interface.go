package ocr2vrf

import (
	"go.uber.org/multierr"

	offchainreporting "github.com/smartcontractkit/libocr/offchainreporting2"

	"github.com/smartcontractkit/ocr2vrf/internal/dkg"
	"github.com/smartcontractkit/ocr2vrf/internal/dkg/contract"
	"github.com/smartcontractkit/ocr2vrf/internal/util"
	"github.com/smartcontractkit/ocr2vrf/internal/vrf"
)

type OCR2VRF struct {
	dkg, vrf *offchainreporting.Oracle
}

type (
	EthereumReportSerializer = vrf.EthereumReportSerializer
)

func NewOCR2VRF(a DKGVRFArgs) (*OCR2VRF, error) {
	transceiver := keyTransceiver{a.KeyID, nil}
	dkg, err := offchainreporting.NewOracle(offchainreporting.OracleArgs{
		BinaryNetworkEndpointFactory: a.BinaryNetworkEndpointFactory,
		V2Bootstrappers:              a.V2Bootstrappers,
		ContractConfigTracker:        a.DKGContractConfigTracker,
		ContractTransmitter:          a.DKGContractTransmitter,
		Database:                     a.DKGDatabase,
		LocalConfig:                  a.DKGLocalConfig,
		Logger:                       a.DKGLogger,
		MonitoringEndpoint:           a.DKGMonitoringEndpoint,
		OffchainConfigDigester:       a.DKGOffchainConfigDigester,
		OffchainKeyring:              a.OffchainKeyring,
		OnchainKeyring:               a.OnchainKeyring,
		ReportingPluginFactory: dkg.NewReportingPluginFactory(
			a.Esk,
			a.Ssk,
			a.KeyID,
			a.DKGContract,
			a.DKGLogger,
			&transceiver,
		),
	})
	if err != nil {
		return nil, util.WrapError(err, "while setting up new DKG oracle")
	}
	confirmationDelays := make(map[uint32]struct{}, len(a.ConfirmationDelays))
	for _, d := range a.ConfirmationDelays {
		confirmationDelays[d] = struct{}{}
	}
	vrf, err := offchainreporting.NewOracle(offchainreporting.OracleArgs{
		BinaryNetworkEndpointFactory: a.BinaryNetworkEndpointFactory,
		V2Bootstrappers:              a.V2Bootstrappers,
		ContractConfigTracker:        a.VRFContractConfigTracker,
		ContractTransmitter:          a.VRFContractTransmitter,
		Database:                     a.VRFDatabase,
		LocalConfig:                  a.VRFLocalConfig,
		Logger:                       a.VRFLogger,
		MonitoringEndpoint:           a.VRFMonitoringEndpoint,
		OffchainConfigDigester:       a.VRFOffchainConfigDigester,
		OffchainKeyring:              a.OffchainKeyring,
		OnchainKeyring:               a.OnchainKeyring,
		ReportingPluginFactory: vrf.NewVRFReportingPluginFactory(
			&transceiver,
			a.Coordinator,
			a.Blockhashes,
			a.Serializer,
			a.VRFLogger,
			a.JulesPerFeeCoin,
			confirmationDelays,
		),
	})
	if err != nil {
		return nil, util.WrapError(err, "while setting up VRF oracle")
	}
	return &OCR2VRF{dkg, vrf}, nil
}

func OffchainConfig(keyID contract.KeyID) []byte {
	return vrf.OffchainConfig(keyID)
}

func OnchainConfig(confDelays map[uint32]struct{}) []byte {
	return vrf.OnchainConfig(confDelays)
}

func (o *OCR2VRF) Start() error {
	if err := o.dkg.Start(); err != nil {
		return util.WrapError(err, "starting DKG oracle")
	}
	if err := util.WrapError(o.vrf.Start(), "starting VRF oracle"); err != nil {
		return multierr.Append(err, util.WrapError(
			o.dkg.Close(),
			"closing DKG process after starting VRF process failed",
		))
	}
	return nil
}

func (o *OCR2VRF) Close() error {
	return multierr.Append(
		util.WrapError(o.dkg.Close(), "while closing DKG process"),
		util.WrapError(o.vrf.Close(), "while closing VRF process"),
	)
}
