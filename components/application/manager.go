package application
//
//type ApplicationManager struct {
//	Config map[string]interface{}
//	Env string
//	Version string
//}
//
//func NewApplicationManager() (*ApplicationManager,error){
//	s := &ApplicationManager{}
//	return  s,nil
//}
//
//func (a ApplicationManager) RegisterStreamInterceptors(...interface{}) {
//	panic("implement me")
//}
//
//func (a ApplicationManager) RegisterUnaryInterceptors(...interface{}) {
//	panic("implement me")
//}
//
//func (a ApplicationManager) RegisterCb(...interface{}) {
//	panic("implement me")
//}
//
//func (a ApplicationManager) NewServ() error {
//	panic("implement me")
//}
//
//func (a ApplicationManager) StartServ() error {
//	panic("implement me")
//}
//
//func (a ApplicationManager) Stop() {
//	panic("implement me")
//}
//
//func (a ApplicationManager) GracefulStop() {
//	panic("implement me")
//}
//
//func (a ApplicationManager) GetServicesInfo() map[string]service_register.ServerInfo {
//	panic("implement me")
//}
//
//func (a ApplicationManager) GetAddress() *net.TCPAddr {
//	panic("implement me")
//}
//
//func (a ApplicationManager) GetListener() *net.TCPListener {
//	panic("implement me")
//}