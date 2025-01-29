package main

import pb "distributed-cache/gen/v1"

type PeerPicker interface {
	PickPeer(key string) (peer PeerGetter, ok bool)
}

type PeerGetter interface {
	Get(in *pb.GetRequest, out *pb.GetResponse) error
}
