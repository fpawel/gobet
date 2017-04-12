package replaceOrder


type replaceInstruction struct{
	// Unique identifier for the bet
	betId  int64

	// The price to replace the bet at
	newPrice float32
}
