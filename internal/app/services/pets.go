package services

import (
	"context"
	"errors"
	"sync"

	pb "github.com/glebnaz/kafka-webinars-rebrain/pkg/pb/api/v1"
)

type Store struct {
	data map[string]*pb.Pet
	m    sync.Mutex
}

func (s *Store) Put(id string, val *pb.Pet) {
	s.m.Lock()

	defer s.m.Unlock()

	s.data[id] = val
}

func (s *Store) Get(id string) (bool, *pb.Pet) {
	s.m.Lock()
	defer s.m.Unlock()
	d, ok := s.data[id]

	return ok, d
}

func (s *Store) Delete(id string) {
	s.m.Lock()

	defer s.m.Unlock()

	delete(s.data, id)
}

type Service struct {
	store Store
}

func (s *Service) GetPet(ctx context.Context, request *pb.GetPetRequest) (*pb.GetPetResponse, error) {
	id := request.GetPetId()

	ok, pet := s.store.Get(id)
	if !ok {
		return nil, errors.New("pet not found")
	}

	return &pb.GetPetResponse{Pet: pet}, nil
}

func (s *Service) PutPet(ctx context.Context, request *pb.PutPetRequest) (*pb.PutPetResponse, error) {

	pet := &pb.Pet{
		Name:    request.GetName(),
		PetId:   request.GetId(),
		PetType: request.GetPetType(),
	}

	s.store.Put(request.GetId(), pet)

	return &pb.PutPetResponse{Pet: pet}, nil
}

func (s *Service) DeletePet(ctx context.Context, request *pb.DeletePetRequest) (*pb.DeletePetResponse, error) {
	panic("implement me")
}

func NewService() *Service {
	d := make(map[string]*pb.Pet)
	s := Store{
		data: d,
	}
	return &Service{store: s}
}
