package main

import (
	"fmt"
	"time"
)

type Player struct {
	x, y float32
}

type Direction struct {
	x, y float32
}

type Game struct {
	manager *UDPManager
	Client  *UDPClient //客户端
	Player1 Player
	mov1    Direction
}

func NewGame(manager *UDPManager, p1 *UDPClient) *Game {
	return &Game{manager, p1, Player{0, 0}, Direction{0, 0}}
}

func (this *Game) AddOperation(p *UDPClient, op *Operation) {
	if op.Type == 1 {
		this.mov1 = op.Object.(Direction)
	}
}

func (this *Game) Run() {
	for {
		if this.Client.LastRecv.Add(time.Second).Before(time.Now()) {
			break
		}
		time.Sleep(time.Millisecond * 15)
		var speed float32 = 0.1
		this.Player1.x += this.mov1.x * speed
		if this.Player1.x > 2 {
			this.Player1.x = 2
		}
		if this.Player1.x < -2 {
			this.Player1.x = -2
		}
		this.Player1.y += this.mov1.y * speed
		if this.Player1.y > 4 {
			this.Player1.y = 4
		}
		if this.Player1.y < -4 {
			this.Player1.y = -4
		}
		this.manager.Send([]*UDPClient{this.Client}, []*Status{NewStatus(1, this.Player1)})
	}
	fmt.Println("玩家已断线!")
}
