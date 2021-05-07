package NewIptables

func upIptablesCap() error {
	//linkPath, err := exec.LookPath("iptables")
	//if err != nil {
	//	logger.Warningf("get iptables path failed, err: %v", err)
	//	return err
	//}
	//finalPath, err := filepath.EvalSymlinks(linkPath)
	//if err != nil {
	//	logger.Warning("get final link path failed, err: %v", err)
	//	return err
	//}
	//
	//inherit := []string{"cap_net_raw", "cap_net_admin"}
	//args := []string{"setcap", strings.Join(inherit, ",") + "=" + "ep", finalPath}
	//

	return nil
}

func downIptablesCap() {

}
