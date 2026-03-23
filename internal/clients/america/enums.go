package america

// DeploymentStatusesType holds the possible deployment statuses.
type DeploymentStatusesType struct {
	PENDING string
	CREATED string
	DELETED string
	ERROR   string
}

// DeploymentStatuses contains the possible deployment statuses.
var DeploymentStatuses = DeploymentStatusesType{
	PENDING: "PENDING",
	CREATED: "CREATED",
	DELETED: "DELETED",
	ERROR:   "ERROR",
}

// AmericaEnvType holds environment constants.
type AmericaEnvType struct {
	GLOBAL string
}

// AmericaEnv contains environment constants.
var AmericaEnv = AmericaEnvType{
	GLOBAL: "GLOBAL",
}

// AmericaRegionsNameType holds region name constants.
type AmericaRegionsNameType struct {
	DEFAULT string
}

// AmericaRegionsName contains region name constants.
var AmericaRegionsName = AmericaRegionsNameType{
	DEFAULT: "DEFAULT",
}
