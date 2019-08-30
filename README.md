Exceeding open file limit for torrent downloads
	Problem - Certain downloads create/open an excess of 1024 files, the UNIX process limit
	Solution -
		Maintain a table of File object entries.
		When the application begins, populate this struct with File objects that describe the file being managed.
		The first a limit above which least recently used (LRU) files are closed. Keeping the number of opened files at the number of
		unchoked peers (i.e. the number of pieces being downloaded, which (not strictly) upper bounds the number of files being opened)
		When a file operation is performed and the threshold is reached. Begin cleanup. Otherwise open the file.

Blog Post (put in github README)
	UML/Hypergraph of processes and synchronized communication (over channels or shared memory)
	Project heirarchy and a description of the functionality of each module and their API
	Address general development, structure and performance issues
		Memory leaks, Networking performance, Multi-threading and Communication

Optmisations
	HAVE suppression
	Profile the code to find the bottlenecks
	If the peer is a seeder, calculating client-peer piece intersection is costly for every block request
	Avoid keeping block data in memory (a faster store than the FS, somehow?)
	MMaped read and write operations
	Fast checksum calculations ? (do these take long?)
	FAST extension - reduced protocol overhead
	Reduce disk overhead
	Update the left tracker statistic
	Only update the optimisitic unchoke every 30 seconds (not 10 seconds)
	Correnctly transition into seeding - update tracker, avoid "should be interested" calculations on bitfield/have messages.
	private trackers ?
	Multi-thread the initial piece checks since SHA1 calculations are slow
	Supervisor model - Erlang (http://4.bp.blogspot.com/_ZmTZzIup5tY/S0OkTpVs7TI/AAAAAAAAACA/h9WjWJahvMI/s1600-h/PHierachy.png)
	Move the disk code in the piece module

Client - start, restart, stop
Supervisor calls these to manage the child according to its child specification, restart strategy and restart intensity


Test in a "lab enviornment" i.e.
	Set up a (private?) tracker for a torrent
	Create a torrent.
	Start seeding the torrent from one computer (use the rtorrent CLI client)
	Log everything to see if it downloading from a single peer works.
	Start another seeder - see if the client cooperates piece downloads.
	Start several leechers - it will start downloading from you!
	See if the client unchokes/chokes properly.

How is the listener going to work without the firewall disabled ? well thats for people who want to use it to figure out!
